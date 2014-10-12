package broker

type flightMessage struct {
	Qos          uint8 // the Qos in effect
	DestClientId string
	Status       uint8 // The status of this message, like PENDING_PUB(deliver occured
	// when client if offline), PENDING_ACK, etc
	MessageInternalId uint64 // The MqttMessage of interest
	ClientMessageId   uint16 // The message id to be used in MQTT packet
}
