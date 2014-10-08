package mqtt

const (
	CONNECT = uint8(iota + 1)
	CONNACK
	PUBLISH
	PUBACK
	PUBREC  // publish received (assured delivery part 1)
	PUBREL  // publish release (assured delivery part 2)
	PUBCOMP // publish complete (assured delivery part 3)
	SUBSCRIBE
	SUBACK
	UNSUBSCRIBE
	UNSUBACK
	PINGREQ // for keepalive
	PINGRESP
	DISCONNECT
)

const (
	SEND_WILL = uint8(iota)
	DONT_SEND_WILL
)

const (
	PENDING_PUB = uint8(iota + 1)
	PENDING_ACK
)

const (
	ACCEPTED = uint8(iota)
	UNACCEPTABLE_PROTOCOL_VERSION
	IDENTIFIER_REJECTED
	SERVER_UNAVAILABLE
	BAD_USERNAME_OR_PASSWORD
	NOT_AUTHORIZED
)
