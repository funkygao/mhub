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
	protocolName      = "MQIsdp"
	protocolVersion   = 3
	maxClientIdLength = 23
)
