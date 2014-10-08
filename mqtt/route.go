package mqtt

import (
	log "github.com/funkygao/log4go"
	"net"
)

var (
	routings = map[uint8]Handler{
		CONNECT:    HandleConnect,
		PUBLISH:    HandlePublish,
		SUBSCRIBE:  HandleSubscribe,
		PINGREQ:    HandlePingreq,
		DISCONNECT: HandleDisconnect,
	}
)

func HandleCommand(mqtt *Mqtt, conn *net.Conn, client **Client) {
	handler, present := routings[mqtt.FixedHeader.MessageType]
	if !present {
		log.Error("unfound command: %v", *mqtt.FixedHeader)
		return
	}

	handler(mqtt, conn, client)
}
