package broker

const clientQueueLength = 100
const sendingQueueLength = 100
const peersQueueLength = 100

// The length of the queue that subscription processing
// workers are taking from.
const postQueue = 100

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
