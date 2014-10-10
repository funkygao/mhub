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
	svr        *Server
	conn       net.Conn
	jobs       chan job
	lastOpTime int64 // // Last Unix timestamp when recieved message from this client
	clientid   string
}

func (c *incomingConn) String() string {
	return c.clientid + "@" + c.conn.RemoteAddr().String()
}

func (c *incomingConn) refreshOpTime() {
	atomic.StoreInt64(&c.lastOpTime, time.Now().Unix())
}

func (c *incomingConn) heartbeat(interval time.Duration) {
	if interval == 0 {
		// disabled
		return
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			deadline := int64(float64(c.lastOpTime) + float64(interval)*1.5)
			if deadline < time.Now().Unix() {
				// ForceDisconnect(client, G_clients_lock, SEND_WILL) TODO
				log.Warn("client(%s) idle too long, kicked out", *c)
			}
		}
	}

}

func (c *incomingConn) add() *incomingConn {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	existing, present := clients[c.clientid]
	if present {
		return existing
	}

	clients[c.clientid] = c
	return nil
}

// Delete a connection; the conection must be closed by the caller first.
func (c *incomingConn) del() {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	delete(clients, c.clientid)
}

// Queue a message; no notification of sending is done.
func (c *incomingConn) submit(m proto.Message) {
	select {
	case c.jobs <- job{m: m}:
	default:
		log.Error("%+v: failed to submit message", *c)
	}
	return
}

// Queue a message, returns a channel that will be readable
// when the message is sent.
func (c *incomingConn) submitSync(m proto.Message) receipt {
	j := job{m: m, r: make(receipt)}
	c.jobs <- j
	return j.r
}

func (c *incomingConn) inboundLoop() {
	defer func() {
		c.svr.stats.clientDisconnect()

		log.Debug("Closed client %s", c)

		c.conn.Close()
		close(c.jobs) // outbound loop will terminate
	}()

	for {
		// TODO: timeout (first message and/or keepalives)
		m, err := proto.DecodeOneMessage(c.conn, nil)
		if err != nil {
			if err != io.EOF {
				log.Error("%v: %s", err, c)
			}

			return
		}

		c.svr.stats.messageRecv()
		c.refreshOpTime()

		if c.svr.cf.Echo {
			log.Debug("%s -> %T %+v", c, m, m)
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
			c.clientid = m.ClientId

			// Disconnect existing connections.
			if existing := c.add(); existing != nil {
				disconnect := &proto.Disconnect{}
				existing.submitSync(disconnect).wait()
				existing.del()
			}
			c.add()

			go c.heartbeat(time.Duration(m.KeepAliveTimer) * time.Second)

			// TODO: Last will
			if !m.CleanSession {
				// deliver flying messages TODO
				// deliver on connect
				// restore client's subscriptions
			}

			c.submit(&proto.ConnAck{
				ReturnCode: rc,
			})

			// close connection if it was a bad connect
			if rc != proto.RetCodeAccepted {
				log.Error("%v: %s", proto.ConnectionErrors[rc], c)
				return
			}

			log.Debug("New client %s (c^%v, k^%v)",
				c, m.CleanSession, m.KeepAliveTimer)

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
				c.svr.subs.submit(c, m)
			}

			c.submit(&proto.PubAck{MessageId: m.MessageId})

		case *proto.PingReq:
			c.submit(&proto.PingResp{})

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
				c.svr.subs.add(tq.Topic, c)

				suback.TopicsQos[i] = proto.QosAtMostOnce
			}
			c.submit(suback)

			// Process retained messages.
			for _, tq := range m.Topics {
				c.svr.subs.sendRetain(tq.Topic, c)
			}

		case *proto.Unsubscribe:
			for _, t := range m.Topics {
				c.svr.subs.unsub(t, c)
			}

			c.submit(&proto.UnsubAck{MessageId: m.MessageId})

		case *proto.Disconnect:
			return

		default:
			log.Error("inbound: unknown msg type %T", m)
			return
		}
	}
}

func (c *incomingConn) outboundLoop() {
	defer func() {
		// Close connection on exit in order to cause inboundLoop to exit.
		c.conn.Close()
		c.del()
		c.svr.subs.unsubAll(c)
	}()

	for {
		select {
		case job := <-c.jobs:
			if c.svr.cf.Echo {
				log.Debug("%s <- %T %+v", c, job.m, job.m)
			}

			// TODO: write timeout
			err := job.m.Encode(c.conn)
			if job.r != nil {
				// notifiy the sender that this message is sent
				close(job.r)
			}
			if err != nil {
				log.Error(err)
				return
			}

			c.svr.stats.messageSend()

			if _, ok := job.m.(*proto.Disconnect); ok {
				log.Error("writer: sent disconnect message")
				return
			}
		}
	}

}
