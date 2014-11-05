package broker

const (
	retainFalse = false
	retainTrue  = true
	dupFalse    = false
	dupTrue     = true
)

const (
	initClientNum = 10000
)

const (
	SLASH               = "/"
	PRESENCE_KEY_PREFIX = "_pre:"
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
