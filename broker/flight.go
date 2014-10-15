package broker

import (
	"sync"
)

type flightMessage struct {
	Qos               uint8 // the Qos in effect
	DestClientId      string
	Status            uint8
	MessageInternalId uint64 // The MqttMessage of interest
	ClientMessageId   uint16 // The message id to be used in MQTT packet
}

type flights struct {
	mu sync.Mutex
}

func newFlights() *flights {
	return &flights{}
}

func (this *flights) getFlyingMessages(clientId string) {

}

func (this *flights) setFlyingMessage(clientId string, messages map[uint16]flightMessage) {

}

func (this *flights) resetFlyingMessages(clientId string) {

}
