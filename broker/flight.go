package broker

type flightMessage struct {
	Qos          uint8 // the Qos in effect
	DestClientId string
	Status       uint8 
	MessageInternalId uint64 // The MqttMessage of interest
	ClientMessageId   uint16 // The message id to be used in MQTT packet
}
