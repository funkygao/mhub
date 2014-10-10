package broker

import (
	"errors"
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
	"net"
	"sync"
)

type peer struct {
	host string
	conn net.Conn
	jobs chan job
}

func newPeer(host string) (this *peer) {
	return &peer{
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

	log.Debug("%+v started", *this)

	for job := range this.jobs {
		err = job.m.Encode(this.conn) // replicated to peer
		if err != nil {
			log.Error(err)
			return
		}
	}

}

func (this *peer) submit(m proto.Message) {
	this.jobs <- job{m: m}
}

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
		this.nodes[node] = newPeer(node)
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

			go this.recvReplication(conn)
		}
	}()

	return nil
}

func (this *peers) discover() {
	log.Debug("discovering...")

}

func (this *peers) recvReplication(conn net.Conn) {
	defer conn.Close()

	log.Debug("got a replicator: %s", conn.RemoteAddr().String())

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

	this.nodes[host] = newPeer(host)
	return nil
}

func (this *peers) leave(host string) {

}

func (this *peers) submit(m proto.Message) {
	for _, p := range this.nodes {
		p.submit(m)
	}
}
