package mqtt

var (
	routings = map[uint8]Handler{
		CONNECT:    HandleConnect,
		PUBLISH:    HandlePublish,
		SUBSCRIBE:  HandleSubscribe,
		PINGREQ:    HandlePingreq,
		DISCONNECT: HandleDisconnect,
	}
)
