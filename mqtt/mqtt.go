package mqtt

type MqttHeader struct {
	MessageType uint8
	DupFlag     bool
	Retain      bool
	QosLevel    uint8
	Lenth       uint32
}

type mqtt struct {
}
