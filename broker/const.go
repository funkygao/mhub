package broker

// TODO mv to config
const clientQueueLength = 100

const (
	retainFalse = false
	retainTrue  = true
	dupFalse    = false
	dupTrue     = true
)

const (
	pendingPub = uint8(iota + 1) // occured when client is offline
	pendingAck                   // occured before client sendback PubAck
)

const (
	protocolName      = "MQIsdp"
	protocolVersion   = 3
	maxClientIdLength = 23
)
