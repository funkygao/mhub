package broker

import (
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
	"io"
	"net"
	"sync/atomic"
	"time"
)

// An incomingConn represents a MQTT connection into a Server.
type incomingConn struct {
	server *Server

	flag proto.Connect

	conn          net.Conn
	jobs          chan job
	heartbeatStop chan struct{}
	lastOpTime    int64 // // Last Unix timestamp when recieved message from this conn
}

func (this *incomingConn) String() string {
	return this.flag.ClientId + "@" + this.conn.RemoteAddr().String()
}

func (this *incomingConn) refreshOpTime() {
	atomic.StoreInt64(&this.lastOpTime, time.Now().Unix())
}

func (this *incomingConn) heartbeat(keepAliveTimer time.Duration) {
	ticker := time.NewTicker(keepAliveTimer)
	defer func() {
		ticker.Stop()
		log.Debug("%s hearbeat stopped", this)
	}()

	for {
		select {
		case <-ticker.C:
			// 1.5*KeepAliveTimer latency tolerance
			deadline := int64(float64(this.lastOpTime) + keepAliveTimer.Seconds()*1.5)
			overIdle := time.Now().Unix() - deadline
			if overIdle > 0 {
				this.submitSync(&proto.Disconnect{}).wait()
				log.Warn("%s over idle %ds, kicked out", this, overIdle)

				if this.flag.WillFlag {
					// TODO broker will publish a message on behalf of the client
				}

				return
			}

		case <-this.heartbeatStop:
			return
		}
	}

}

func (this *incomingConn) add() *incomingConn {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	existing, present := clients[this.flag.ClientId]
	if present {
		return existing
	}

	clients[this.flag.ClientId] = this
	return nil
}

// Delete a connection; the conection must be closed by the caller first.
func (this *incomingConn) del() {
	clientsMu.Lock()
	delete(clients, this.flag.ClientId)
	clientsMu.Unlock()
}

// Queue a message; no notification of sending is done.
func (this *incomingConn) submit(m proto.Message) {
	select {
	case this.jobs <- job{m: m}:
	default:
		log.Error("%s: failed to put in jobs chan", this)
	}
}

// Queue a message, returns a channel that will be readable
// when the message is sent.
func (this *incomingConn) submitSync(m proto.Message) receipt {
	j := job{m: m, r: make(receipt)}
	this.jobs <- j
	return j.r
}

// race:
// client disconnect -> inboundLoop close jobs ->
// outboundLoop stops -> outboundLoop close conn, unsubAll, del
func (this *incomingConn) inboundLoop() {
	defer func() {
		this.server.stats.clientDisconnect()

		close(this.jobs) // will terminate outboundLoop
	}()

	for {
		// TODO: timeout
		m, err := proto.DecodeOneMessage(this.conn, nil)
		if err != nil {
			if err != io.EOF {
				// e,g. connection reset by peer
				log.Error("%v: %s", err, this)
			}

			return
		}

		this.server.stats.messageRecv()
		this.refreshOpTime()

		if this.server.cf.Broker.Echo {
			log.Debug("%s -> %T %+v", this, m, m)
		}

		switch m := m.(type) {
		case *proto.Connect: // TODO close conn if too long no Connect
			rc := proto.RetCodeAccepted

			// validate protocol name and version
			if m.ProtocolName != protocolName ||
				m.ProtocolVersion != protocolVersion {
				log.Error("invalid connection[%s] protocol %s, version %d",
					this,
					m.ProtocolName, m.ProtocolVersion)
				rc = proto.RetCodeUnacceptableProtocolVersion
			}

			// validate client id length
			if len(m.ClientId) < 1 || len(m.ClientId) > maxClientIdLength {
				rc = proto.RetCodeIdentifierRejected
			}
			this.flag = *m // connection flag

			// Disconnect existing connections.
			if existing := this.add(); existing != nil {
				log.Warn("found dup client: %s", this)

				disconnect := &proto.Disconnect{}
				existing.submitSync(disconnect).wait()
				existing.del()
			}
			this.add()

			if m.KeepAliveTimer > 0 {
				go this.heartbeat(time.Duration(m.KeepAliveTimer) * time.Second)
			}

			// TODO: Last will
			// The will option allows clients to prepare for the worst.
			if !m.CleanSession {
				// broker will keep the subscription active even after the client disconnects
				// It will also queue any new messages it receives for the client, but
				// only if they have QoS>0
				// restore client's subscriptions
				// deliver flying messages TODO
				// deliver on connect
			}

			this.submit(&proto.ConnAck{ReturnCode: rc})

			// close connection if it was a bad connect
			if rc != proto.RetCodeAccepted {
				log.Error("%v: %s", proto.ConnectionErrors[rc], this)
				return
			}

			log.Debug("new client: %s (c^%v, k^%v)",
				this, m.CleanSession, m.KeepAliveTimer)

		case *proto.Publish:
			// TODO assert m.TopicName is not wildcard

			// replicate message to all subscribers of this topic
			this.server.subs.submit(m)

			// replication to peers
			this.server.peers.submit(m)

			// for QoS 0, we need do nothing
			if m.Header.QosLevel == proto.QosAtLeastOnce { // QoS 1
				if m.MessageId == 0 {
					log.Error("client[%s] invalid message id", this)
				}

				this.submit(&proto.PubAck{MessageId: m.MessageId})
			}

		case *proto.Subscribe:
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

			// Process retained messages
			for _, tq := range m.Topics {
				this.server.subs.sendRetain(tq.Topic, this)
			}

		case *proto.Unsubscribe:
			for _, t := range m.Topics {
				this.server.subs.unsub(t, this)
			}

			this.submit(&proto.UnsubAck{MessageId: m.MessageId})

		case *proto.PingReq:
			this.submit(&proto.PingResp{})

		case *proto.Disconnect:
			log.Debug("%s actively disconnect", this)
			return

		case *proto.PubAck:
			// get flying messages for this client
			// if not found, ignore this PubAck
			// if found, mark this flying message

		default:
			log.Warn("%s -> unexpected %T", this, m)
			return
		}
	}
}

func (this *incomingConn) outboundLoop() {
	defer func() {
		// close connection on exit in order to cause inboundLoop to exit.
		// only outboundLoop can close conn,
		// otherwise outboundLoop will error: use of closed network connection
		log.Debug("%s conn closed", this)
		close(this.heartbeatStop)
		this.conn.Close()

		this.del()
		this.server.subs.unsubAll(this)
	}()

	for {
		select {
		case job, on := <-this.jobs:
			if !on {
				// jobs chan was closed by inboundLoop
				return
			}

			if this.server.cf.Broker.Echo {
				log.Debug("%s <- %T %+v", this, job.m, job.m)
			}

			// TODO: write timeout
			err := job.m.Encode(this.conn)
			if job.r != nil {
				// notifiy the sender that this message is sent
				close(job.r)
			}
			if err != nil {
				log.Error("%s %s", this, err)
				return
			}

			this.server.stats.messageSend()

			if _, ok := job.m.(*proto.Disconnect); ok {
				return
			}
		}
	}

}
