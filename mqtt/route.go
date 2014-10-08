package mqtt

import (
	log "github.com/funkygao/log4go"
	"net"
)

var (
	routings = map[uint8]Handler{
		MSG_TYPE_CONNECT:    HandleConnect,
		MSG_TYPE_PUBLISH:    HandlePublish,
		MSG_TYPE_SUBSCRIBE:  HandleSubscribe,
		MSG_TYPE_PINGREQ:    HandlePingreq,
		MSG_TYPE_DISCONNECT: HandleDisconnect,
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
