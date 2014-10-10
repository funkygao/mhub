package broker

import (
	"errors"
	log "github.com/funkygao/log4go"
	"github.com/funkygao/mhub/config"
	proto "github.com/funkygao/mqttmsg"
	"net"
	"sync"
)

type peers struct {
	nodes  map[string]*peer // key is hostname
	mu     sync.Mutex
	server *Server
}

func newPeers(server *Server) (this *peers) {
	this = new(peers)
	this.server = server
	this.nodes = make(map[string]*peer)
	return
}

func (this *peers) start(listenAddr string) error {
	go this.discover()

	// add self to peers for testing, TODO kill this
	if true {
		node := "localhost:9090"
		this.nodes[node] = newPeer(node, this.server.cf.Peers)
		go this.nodes[node].start()
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Error(err)
				continue
			}

			this.server.stats.peerConnect()
			log.Debug("peer[%s] accepted", conn.RemoteAddr().String())

			go this.recvReplication(conn)
		}
	}()

	return nil
}

func (this *peers) discover() {
	log.Debug("discovering...")

	// only after startup server, will it register to etcd
	// otherwise, losing data

}

func (this *peers) recvReplication(conn net.Conn) {
	defer func() {
		conn.Close()

		this.server.stats.peerDisconnect()
	}()

	for {
		m, err := proto.DecodeOneMessage(conn, nil)
		if err != nil {
			log.Error(err)
			return
		}

		log.Debug("a message for replication: %+v", m)

		p, ok := m.(*proto.Publish)
		if !ok {
			log.Error("only PUBLISH is replicated, got %+v", m)
			continue
		}

		this.server.subs.submit(p)
	}

}

func (this *peers) join(host string) error {
	this.mu.Lock()
	defer this.mu.Unlock()

	if _, present := this.nodes[host]; present {
		return errors.New("peer already exists")
	}

	this.nodes[host] = newPeer(host, this.server.cf.Peers)
	return nil
}

func (this *peers) leave(host string) {

}

func (this *peers) submit(m proto.Message) {
	for _, p := range this.nodes {
		p.submit(m)
	}
}

type peer struct {
	cf   config.PeersConfig
	host string
	conn net.Conn // outbound conn
	jobs chan job
}

func newPeer(host string, cf config.PeersConfig) (this *peer) {
	return &peer{
		cf:   cf,
		host: host,
		jobs: make(chan job, peersQueueLength),
	}
}

func (this *peer) start() {
	defer func() {
		this.conn.Close()
		close(this.jobs)
	}()

	var err error
	this.conn, err = net.Dial("tcp", this.host)
	if err != nil {
		log.Error(err)
		return
	}

	tcpConn, _ := this.conn.(*net.TCPConn)
	tcpConn.SetNoDelay(this.cf.TcpNoDelay)
	tcpConn.SetKeepAlive(this.cf.Keepalive)

	log.Debug("peer[%+v] connected", this.host)

	for job := range this.jobs {
		err = job.m.Encode(this.conn) // replicated to peer
		if err != nil {
			log.Error(err)
			return
		}
	}

}

func (this *peer) submit(m proto.Message) {
	// TODO send on closed channel
	// the principle is: senders close; receivers check for closed
	this.jobs <- job{m: m}
}
