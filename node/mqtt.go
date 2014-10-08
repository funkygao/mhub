package node

import (
	"github.com/funkygao/gomqtt/mqtt"
	log "github.com/funkygao/log4go"
	"net"
	"runtime/debug"
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

		go this.handleConnection(&conn)
	}
}

func (this *Node) handleConnection(conn *net.Conn) {
	//log.Debug("got conn: %s", conn.(*net.TCPConn).RemoteAddr().String())

	var client *mqtt.Client = nil

	defer func() {
		//log.Debug("close conn: %s", conn.RemoteAddr().String())
		if r := recover(); r != nil {
			debug.PrintStack()
		}

		if client != nil {
			// force client disconnect
		}

		//conn.Close()
	}()

	for {
		// read fixed header of MQTT
		fixedHeader, body := mqtt.ReadCompleteCommand(conn)
		if fixedHeader == nil {
			//log.Error("[%s] nil fixed header", conn.RemoteAddr().String())
			return
		}

		parsed, err := mqtt.DecodeAfterFixedHeader(fixedHeader, body)
		if err != nil {
			log.Error(err)
			continue
		}

		mqtt.HandleCommand(parsed, conn, &client)

	}

}
