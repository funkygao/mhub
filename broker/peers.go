package broker

import (
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
	"net"
	"sync"
)

type peers struct {
	server *Server

	nodes map[string]*endpoint // key is hostname or ip
	mu    sync.Mutex
}

func newPeers(server *Server) (this *peers) {
	this = new(peers)
	this.server = server
	this.nodes = make(map[string]*endpoint)
	return
}

// racing:
// broker listener ready -> peer listener ready -> register broker presence
func (this *peers) start(listenAddr string) error {
	go this.discover()

	// add self to peers for testing, TODO kill this
	// FIXME if true, will lead to send on closed channel err
	if true {
		host := "localhost:9090"
		this.nodes[host] = newEndpoint(host, this.server.cf.Peers, this.server.stats)
		go this.nodes[host].start()
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	log.Info("Accepting peers conn on %s", listenAddr)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Error(err)
				continue
			}

			this.server.stats.peerConnect()
			log.Debug("peer[%s] accepted", conn.RemoteAddr())

			go this.recvReplication(conn)
		}
	}()

	return nil
}

func (this *peers) discover() {
	log.Debug("discovering peers...")

	// only after startup server, will it register to etcd
	// otherwise, losing data

}

func (this *peers) recvReplication(conn net.Conn) {
	defer func() {
		conn.Close()
		log.Warn("peer self die")
		this.server.stats.peerDisconnect()
	}()

	for {
		m, err := proto.DecodeOneMessage(conn, nil)
		if err != nil {
			log.Error(err)
			return
		}

		if this.server.cf.Peers.Echo {
			log.Debug("peers <- %T %+v", m, m)
		}

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
		return errEndpointDupJoin
	}

	this.nodes[host] = newEndpoint(host, this.server.cf.Peers, this.server.stats)
	return nil
}

func (this *peers) leave(host string) {

}

func (this *peers) submit(m proto.Message) {
	for _, p := range this.nodes {
		p.submit(m)
	}
}
