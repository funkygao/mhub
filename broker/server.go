package broker

import (
	"crypto/tls"
	"github.com/funkygao/gomqtt/config"
	log "github.com/funkygao/log4go"
	"net"
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
func NewServer(cf *config.Config) (this *Server) {
	s := &stats{interval: cf.StatsInterval}
	this = &Server{
		cf:    cf,
		stats: s,
		Done:  make(chan struct{}),
		subs:  newSubscriptions(cf.BroadcastWorkers, s),
	}

	go this.stats.start()

	return
}

// Start makes the Server start accepting and handling connections.
func (this *Server) Start() {
	listener, err := this.startListener()
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

			this.stats.clientConnect()

			client := &incomingConn{
				server: this,
				conn:   conn,
				jobs:   make(chan job, sendingQueueLength),
			}
			go client.inboundLoop()
			go client.outboundLoop()
		}
	}()
}

func (this *Server) Stop() {
	close(this.Done)
}

func (this *Server) startListener() (listener net.Listener, err error) {
	if this.cf.ListenAddr != "" {
		listener, err = net.Listen("tcp", this.cf.ListenAddr)
		return
	}

	// TLS
	var cert tls.Certificate
	cert, err = tls.LoadX509KeyPair(this.cf.TlsServerCert, this.cf.TlsServerKey)
	if err != nil {
		return
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"mqtt"},
	}
	listener, err = tls.Listen("tcp", this.cf.TlsListenAddr, cfg)
	return
}
