package broker

import (
	proto "github.com/funkygao/mqttmsg"
)

type persist interface {
	AddFlightMessage(clientId string, message proto.Message)
	DelFlightMessage(clientId string, messageId uint16)        // key is messageId
	GetFlightMessage(clientId string) map[uint16]proto.Message // key is messageId
}
