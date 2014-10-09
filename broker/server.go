package broker

import (
	"crypto/tls"
	"fmt"
	"github.com/funkygao/gomqtt/config"
	proto "github.com/funkygao/mqttmsg"
	"io"
	"log"
	"math/rand"
	"net"
	"runtime"
	"strings"
	"time"
)

// A Server holds all the state associated with an MQTT server.
type Server struct {
	cf            *config.Config
	listener      net.Listener
	subs          *subscriptions
	stats         *stats
	Done          chan struct{}
	StatsInterval time.Duration
	Dump          bool // When true, dump the messages in and out.
	rand          *rand.Rand
}

// NewServer creates a new MQTT server, which accepts connections from
// the given listener.
func NewServer(cf *config.Config) *Server {
	svr := &Server{
		cf:            cf,
		stats:         &stats{},
		Done:          make(chan struct{}),
		StatsInterval: time.Second * 10,
		subs:          newSubscriptions(runtime.GOMAXPROCS(0)),
	}

	// start the stats reporting goroutine
	go func() {
		for {
			svr.stats.publish(svr.subs, svr.StatsInterval)

			select {
			case <-svr.Done:
				return
			default:
				// keep going
			}
			time.Sleep(svr.StatsInterval)
		}
	}()

	return svr
}

// Start makes the Server start accepting and handling connections.
func (s *Server) Start() {
	s.startListener()

	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				log.Println(err)
				break
			}

			client := s.newIncomingConn(conn)
			s.stats.clientConnect()
			client.start()
		}

		close(s.Done)
	}()
}

// newIncomingConn creates a new incomingConn associated with this
// server. The connection becomes the property of the incomingConn
// and should not be touched again by the caller until the Done
// channel becomes readable.
func (s *Server) newIncomingConn(conn net.Conn) *incomingConn {
	return &incomingConn{
		svr:  s,
		conn: conn,
		jobs: make(chan job, sendingQueueLength),
		Done: make(chan struct{}),
	}
}

func (s *Server) Stop() {
	close(s.Done)
}

func (s *Server) startListener() (err error) {
	if s.cf.ListenAddr != "" {
		s.listener, err = net.Listen("tcp", s.cf.ListenAddr)
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
	s.listener, err = tls.Listen("tcp", s.cf.TlsListenAddr, cfg)
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

// Start reading and writing on this connection.
func (c *incomingConn) start() {
	go c.reader()
	go c.writer()
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
		log.Print(c, ": failed to submit message")
	}
	return
}

func (c *incomingConn) String() string {
	return fmt.Sprintf("{IncomingConn: %v}", c.clientid)
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
			log.Print("reader: ", err)
			return
		}
		c.svr.stats.messageRecv()

		if c.svr.Dump {
			log.Printf("dump  in: %T", m)
		}

		switch m := m.(type) {
		case *proto.Connect:
			rc := proto.RetCodeAccepted

			if m.ProtocolName != protocolName ||
				m.ProtocolVersion != protocolVersion {
				log.Print("reader: reject connection from ", m.ProtocolName, " version ", m.ProtocolVersion)
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
				log.Printf("Connection refused for %v: %v", c.conn.RemoteAddr(), ConnectionErrors[rc])
				return
			}

			// Log in mosquitto format.
			clean := 0
			if m.CleanSession {
				clean = 1
			}
			log.Printf("New client connected from %v as %v (c%v, k%v).", c.conn.RemoteAddr(), c.clientid, clean, m.KeepAliveTimer)

		case *proto.Publish:
			// TODO: Proper QoS support
			if m.Header.QosLevel != proto.QosAtMostOnce {
				log.Printf("reader: no support for QoS %v yet", m.Header.QosLevel)
				return
			}
			if isWildcard(m.TopicName) {
				log.Print("reader: ignoring PUBLISH with wildcard topic ", m.TopicName)
			} else {
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
			log.Printf("reader: unknown msg type %T", m)
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
		if c.svr.Dump {
			log.Printf("dump out: %T", job.m)
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
			log.Print("writer: ", err)
			return
		}
		c.svr.stats.messageSend()

		if _, ok := job.m.(*proto.Disconnect); ok {
			log.Print("writer: sent disconnect message")
			return
		}
	}
}
