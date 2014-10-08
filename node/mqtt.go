package node

import (
	log "github.com/funkygao/log4go"
	"net"
)

func (this *Node) serveMqtt() {
	listener, err := net.Listen("tcp", this.cf.ListenAddr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error(err)
			continue
		}

		go this.handleConnection(conn.(*net.TCPConn))
	}
}

func (this *Node) handleConnection(conn *net.TCPConn) {
	log.Debug("conn: %s", conn.RemoteAddr().String())

}
