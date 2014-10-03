package mqtt

const (
	CONNECT = uint8(iota + 1)
	CONNACK
	PUBLISH
	PUBACK
	PUBREC // publish received (assured delivery part 1)
	PUBREL // publish release (assured delivery part 2)
	PUBCOMP // publish complete (assured delivery part 3)
	SUBSCRIBE
	SUBACK
	UNSUBSCRIBE
	UNSUBACK
	PINGREQ // for keepalive
	PINGRESP
	DISCONNECT
)
