package broker

import (
	log "github.com/funkygao/log4go"
	"github.com/funkygao/mhub/config"
	proto "github.com/funkygao/mqttmsg"
	"net"
	"time"
)

type endpoint struct {
	cf   config.PeersConfig
	host string   // host of other node, not myself
	conn net.Conn // outbound conn to other node
	jobs chan job
}

func newEndpoint(host string, cf config.PeersConfig) (this *endpoint) {
	return &endpoint{
		cf:   cf,
		host: host,
		jobs: make(chan job, cf.QueueLen),
	}
}

func (this *endpoint) start() {
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

	log.Info("peer[%+v] connected", this.host)

	// consume jobs and send to subscription clients
	for job := range this.jobs {
		this.conn.SetWriteDeadline(time.Now().Add(this.cf.IoTimeout))

		err = job.m.Encode(this.conn) // replicated to peer
		if err != nil {
			log.Error(err)
			return
		}
	}

}

func (this *endpoint) submit(m proto.Message) {
	// TODO send on closed channel
	// the principle is: senders close; receivers check for closed
	this.jobs <- job{m: m}
}
