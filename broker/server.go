package broker

import (
	"crypto/tls"
	"github.com/funkygao/gomqtt/config"
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
	"io"
	"net"
	"strings"
)

// A Server holds all the state associated with an MQTT server.
type Server struct {
	cf *config.Config

	stats *stats
	subs  *subscriptions

	Done chan struct{}
}

// NewServer creates a new MQTT server, which accepts connections from
// the given listener.
func NewServer(cf *config.Config) *Server {
	this := &Server{
		cf:    cf,
		stats: &stats{interval: cf.StatsInterval},
		Done:  make(chan struct{}),
		subs:  newSubscriptions(cf.BroadcastWorkers),
	}

	go this.stats.start()

	return this
}

// Start makes the Server start accepting and handling connections.
func (s *Server) Start() {
	listener, err := s.startListener()
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Error(err)
				continue
			}

			s.stats.clientConnect()
			session := &incomingConn{
				svr:  s,
				conn: conn,
				jobs: make(chan job, sendingQueueLength),
				Done: make(chan struct{}),
			}
			go session.reader()
			go session.writer()
		}

		close(s.Done)
	}()
}

func (s *Server) Stop() {
	close(s.Done)
}

func (s *Server) startListener() (listener net.Listener, err error) {
	if s.cf.ListenAddr != "" {
		listener, err = net.Listen("tcp", s.cf.ListenAddr)
		return
	}

	// TLS
	var cert tls.Certificate
	cert, err = tls.LoadX509KeyPair(s.cf.TlsServerCert, s.cf.TlsServerKey)
	if err != nil {
		return
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"mqtt"},
	}
	listener, err = tls.Listen("tcp", s.cf.TlsListenAddr, cfg)
	return
}

// An IncomingConn represents a connection into a Server.
type incomingConn struct {
	svr      *Server
	conn     net.Conn
	jobs     chan job
	clientid string
	Done     chan struct{}
}

// Add this	connection to the map, or find out that an existing connection
// already exists for the same client-id.
func (c *incomingConn) add() *incomingConn {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	existing, ok := clients[c.clientid]
	if !ok {
		// this client id already exists, return it
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
	return
}

// Replace any existing connection with this one. The one to be replaced,
// if any, must be closed first by the caller.
func (c *incomingConn) replace() {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	// Check that any existing connection is already closed.
	existing, ok := clients[c.clientid]
	if ok {
		die := false
		select {
		case _, ok := <-existing.jobs:
			// what? we are expecting that this channel is closed!
			if ok {
				die = true
			}
		default:
			die = true
		}
		if die {
			panic("attempting to replace a connection that is not closed")
		}

		delete(clients, c.clientid)
	}

	clients[c.clientid] = c
	return
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

func (c *incomingConn) reader() {
	// On exit, close the connection and arrange for the writer to exit
	// by closing the output channel.
	defer func() {
		c.conn.Close()
		c.svr.stats.clientDisconnect()
		close(c.jobs)
	}()

	for {
		// TODO: timeout (first message and/or keepalives)
		m, err := proto.DecodeOneMessage(c.conn, nil)
		if err != nil {
			if err == io.EOF {
				return
			}
			if strings.HasSuffix(err.Error(), "use of closed network connection") {
				return
			}
			log.Error(err)
			return
		}
		c.svr.stats.messageRecv()

		if c.svr.cf.Echo {
			log.Debug("dump  in: %T", m)
		}

		switch m := m.(type) {
		case *proto.Connect:
			rc := proto.RetCodeAccepted

			if m.ProtocolName != protocolName ||
				m.ProtocolVersion != protocolVersion {
				log.Error("reader: reject connection from %s, version %d",
					m.ProtocolName, m.ProtocolVersion)
				rc = proto.RetCodeUnacceptableProtocolVersion
			}

			// Check client id.
			if len(m.ClientId) < 1 || len(m.ClientId) > maxClientIdLength {
				rc = proto.RetCodeIdentifierRejected
			}
			c.clientid = m.ClientId

			// Disconnect existing connections.
			if existing := c.add(); existing != nil {
				disconnect := &proto.Disconnect{}
				r := existing.submitSync(disconnect)
				r.wait()
				existing.del()
			}
			c.add()

			// TODO: Last will

			connack := &proto.ConnAck{
				ReturnCode: rc,
			}
			c.submit(connack)

			// close connection if it was a bad connect
			if rc != proto.RetCodeAccepted {
				log.Error("Connection refused for %v: %v",
					c.conn.RemoteAddr(), ConnectionErrors[rc])
				return
			}

			// Log in mosquitto format.
			clean := 0
			if m.CleanSession {
				clean = 1
			}
			log.Debug("New client connected from %v as %v (c%v, k%v).",
				c.conn.RemoteAddr(), c.clientid, clean, m.KeepAliveTimer)

		case *proto.Publish:
			// TODO: Proper QoS support
			if m.Header.QosLevel != proto.QosAtMostOnce {
				log.Error("reader: no support for QoS %v yet", m.Header.QosLevel)
				return
			}
			if isWildcard(m.TopicName) {
				log.Error("reader: ignoring PUBLISH with wildcard topic ", m.TopicName)
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
			ack := &proto.UnsubAck{MessageId: m.MessageId}
			c.submit(ack)

		case *proto.Disconnect:
			return

		default:
			log.Error("reader: unknown msg type %T", m)
			return
		}
	}
}

func (c *incomingConn) writer() {
	// Close connection on exit in order to cause reader to exit.
	defer func() {
		c.conn.Close()
		c.del()
		c.svr.subs.unsubAll(c)
	}()

	for job := range c.jobs {
		if c.svr.cf.Echo {
			log.Debug("dump out: %T", job.m)
		}

		// TODO: write timeout
		err := job.m.Encode(c.conn)
		if job.r != nil {
			// notifiy the sender that this message is sent
			close(job.r)
		}
		if err != nil {
			// This one is not interesting; it happens when clients
			// disappear before we send their acks.
			if err.Error() == "use of closed network connection" {
				return
			}
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
