package node

import (
	"crypto/tls"
	"github.com/funkygao/gomqtt/mqtt"
	log "github.com/funkygao/log4go"
	"net"
	"runtime/debug"
)

func (this *Node) serveMqtt() {
	if this.cf.ListenAddr != "" {
		this.serveMqttPlain()
	}

}

func (this *Node) serveMqttPlain() {
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

func (this *Node) serveMqttTls() {
	cert, err := tls.LoadX509KeyPair(this.cf.TlsServerCert, this.cf.TlsServerKey)
	if err != nil {
		panic(err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"mqtt"},
	}
	listener, err := tls.Listen("tcp", this.cf.TlsListenAddr, cfg)
	if err != nil {
		panic(err)
	}

	for {
		listener.Accept()
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
