package mqtt

const (
	MSG_TYPE_CONNECT = uint8(iota + 1)
	MSG_TYPE_CONNACK
	MSG_TYPE_PUBLISH
	MSG_TYPE_PUBACK
	MSG_TYPE_PUBREC  // publish received (assured delivery part 1)
	MSG_TYPE_PUBREL  // publish release (assured delivery part 2)
	MSG_TYPE_PUBCOMP // publish complete (assured delivery part 3)
	MSG_TYPE_SUBSCRIBE
	MSG_TYPE_SUBACK
	MSG_TYPE_UNSUBSCRIBE
	MSG_TYPE_UNSUBACK
	MSG_TYPE_PINGREQ // for keepalive
	MSG_TYPE_PINGRESP
	MSG_TYPE_DISCONNECT
)

const (
	FIXED_HEADER_SIZE = 2
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

const (
	QOS_AT_MOST_ONCE = uint8(iota)
	QOS_AT_LEAST_ONCE
	QOS_EXACTLY_ONCE
)

const (
	PROTOCOL_NAME = "MQIsdp"
	PROTOCOL_VER  = 3
)
