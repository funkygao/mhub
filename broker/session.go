package broker

import (
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
	"io"
	"net"
	"sync/atomic"
	"time"
)

// An IncomingConn represents a connection into a Server.
type incomingConn struct {
	server     *Server
	conn       net.Conn
	jobs       chan job
	lastOpTime int64 // // Last Unix timestamp when recieved message from this client
	clientid   string
}

func (this *incomingConn) String() string {
	return this.clientid + "@" + this.conn.RemoteAddr().String()
}

func (this *incomingConn) refreshOpTime() {
	atomic.StoreInt64(&this.lastOpTime, time.Now().Unix())
}

func (this *incomingConn) heartbeat(interval time.Duration) {
	if interval == 0 {
		// disabled
		return
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			deadline := int64(float64(this.lastOpTime) + float64(interval)*1.5)
			if deadline < time.Now().Unix() {
				// ForceDisconnect(client, G_clients_lock, SEND_WILL) TODO
				log.Warn("client(%s) idle too long, kicked out", this)
			}
		}
	}

}

func (this *incomingConn) add() *incomingConn {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	existing, present := clients[this.clientid]
	if present {
		return existing
	}

	clients[this.clientid] = this
	return nil
}

// Delete a connection; the conection must be closed by the caller first.
func (this *incomingConn) del() {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	delete(clients, this.clientid)
}

// Queue a message; no notification of sending is done.
func (this *incomingConn) submit(m proto.Message) {
	select {
	case this.jobs <- job{m: m}:
	default:
		log.Error("%s: failed to submit message", this)
	}
	return
}

// Queue a message, returns a channel that will be readable
// when the message is sent.
func (this *incomingConn) submitSync(m proto.Message) receipt {
	j := job{m: m, r: make(receipt)}
	this.jobs <- j
	return j.r
}

func (this *incomingConn) inboundLoop() {
	defer func() {
		this.server.stats.clientDisconnect()

		this.conn.Close()
		close(this.jobs) // outbound loop will terminate

		log.Debug("%s conn closed", this)
	}()

	for {
		// TODO: timeout (first message and/or keepalives)
		m, err := proto.DecodeOneMessage(this.conn, nil)
		if err != nil {
			if err != io.EOF {
				log.Error("%v: %s", err, this)
			}

			return
		}

		this.server.stats.messageRecv()
		this.refreshOpTime()

		if this.server.cf.Echo {
			log.Debug("%s -> %T %+v", this, m, m)
		}

		switch m := m.(type) {
		case *proto.Connect:
			rc := proto.RetCodeAccepted

			// validate protocol name and version
			if m.ProtocolName != protocolName ||
				m.ProtocolVersion != protocolVersion {
				log.Error("inbound: reject connection from %s, version %d",
					m.ProtocolName, m.ProtocolVersion)
				rc = proto.RetCodeUnacceptableProtocolVersion
			}

			// validate client id
			if len(m.ClientId) < 1 || len(m.ClientId) > maxClientIdLength {
				rc = proto.RetCodeIdentifierRejected
			}
			this.clientid = m.ClientId

			// Disconnect existing connections.
			if existing := this.add(); existing != nil {
				disconnect := &proto.Disconnect{}
				existing.submitSync(disconnect).wait()
				existing.del()
			}
			this.add()

			go this.heartbeat(time.Duration(m.KeepAliveTimer) * time.Second)

			// TODO: Last will
			if !m.CleanSession {
				// deliver flying messages TODO
				// deliver on connect
				// restore client's subscriptions
			}

			this.submit(&proto.ConnAck{
				ReturnCode: rc,
			})

			// close connection if it was a bad connect
			if rc != proto.RetCodeAccepted {
				log.Error("%v: %s", proto.ConnectionErrors[rc], this)
				return
			}

			log.Info("New client %s (c^%v, k^%v)",
				this, m.CleanSession, m.KeepAliveTimer)

		case *proto.Publish:
			// TODO support QoS 1
			if m.Header.QosLevel != proto.QosAtMostOnce {
				log.Error("inbound: no support for QoS %v yet", m.Header.QosLevel)
				return
			}

			if isWildcard(m.TopicName) {
				log.Error("inbound: ignoring PUBLISH with wildcard topic ", m.TopicName)
			} else {
				// replicate message to all subscribers of this topic
				this.server.subs.submit(m)

				// replication to peers
				this.server.peers.submit(m)
			}

			this.submit(&proto.PubAck{MessageId: m.MessageId})

		case *proto.PingReq:
			this.submit(&proto.PingResp{})

		case *proto.Subscribe:
			if m.Header.QosLevel != proto.QosAtLeastOnce {
				// protocol error, disconnect
				return
			}

			suback := &proto.SubAck{
				MessageId: m.MessageId,
				TopicsQos: make([]proto.QosLevel, len(m.Topics)),
			}
			for i, tq := range m.Topics {
				// TODO: Handle varying QoS correctly
				this.server.subs.add(tq.Topic, this)

				suback.TopicsQos[i] = proto.QosAtMostOnce
			}
			this.submit(suback)

			// Process retained messages.
			for _, tq := range m.Topics {
				this.server.subs.sendRetain(tq.Topic, this)
			}

		case *proto.Unsubscribe:
			for _, t := range m.Topics {
				this.server.subs.unsub(t, this)
			}

			this.submit(&proto.UnsubAck{MessageId: m.MessageId})

		case *proto.Disconnect:
			return

		default:
			log.Error("inbound: unknown msg type %T", m)
			return
		}
	}
}

func (this *incomingConn) outboundLoop() {
	defer func() {
		// Close connection on exit in order to cause inboundLoop to exit.
		this.conn.Close()
		this.del()
		this.server.subs.unsubAll(this)
	}()

	for {
		select {
		case job, on := <-this.jobs:
			if !on {
				log.Debug("%s jobs closed", this)
				return
			}

			if this.server.cf.Echo {
				log.Debug("%s <- %T %+v", this, job.m, job.m)
			}

			// TODO: write timeout
			err := job.m.Encode(this.conn)
			if job.r != nil {
				// notifiy the sender that this message is sent
				close(job.r)
			}
			if err != nil {
				log.Error(err)
				return
			}

			this.server.stats.messageSend()

			if _, ok := job.m.(*proto.Disconnect); ok {
				log.Error("writer: sent disconnect message")
				return
			}
		}
	}

}
