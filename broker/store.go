package broker

import (
	"fmt"
	proto "github.com/funkygao/mqttmsg"
	"strconv"
)

const (
	_IBOUND_PREFIX = "i."
	_OBOUND_PREFIX = "o."
)

// Store is an interface which can be used to provide implementations
// for message persistence.
// Because we may have to store distinct messages with the same
// messageId, we need a unique key for each message. This is
// possible by prepending "i." or "o." to each message id
type Store interface {
	Open()
	Put(key string, m proto.Message)
	Get(key string) proto.Message
	All() []string
	Del(key string)
	Close()
	Reset()
}

// A key MUST have the form "X.[messageid]"
// where X is 'i' or 'o'
func key2mid(key string) uint16 {
	i, _ := strconv.Atoi(key[2:])
	return uint16(i)
}

// Return a string of the form "i.[id]"
func ibound_mid2key(mid uint16) string {
	return fmt.Sprintf("%s%d", _IBOUND_PREFIX, mid)
}

// Return a string of the form "o.[id]"
func obound_mid2key(mid uint16) string {
	return fmt.Sprintf("%s%d", _OBOUND_PREFIX, mid)
}

// govern which outgoing messages are persisted
func persist_outbound(s Store, m proto.Message) {
	switch m := m.(type) {
	case *proto.Publish:
		if m.Header.QosLevel == proto.QosAtMostOnce {
			s.Del(ibound_mid2key(m.MessageId))
		} else if m.Header.QosLevel == proto.QosAtLeastOnce {
			// store in obound until PubAck received
			s.Put(obound_mid2key(m.MessageId), m)
		}

	case *proto.Subscribe:
		// sending Subscribe, store in obound until SubAck received
		s.Put(obound_mid2key(m.MessageId), m)

	case *proto.Unsubscribe:
		// until UnsubAck received
		s.Put(obound_mid2key(m.MessageId), m)
	}

}

// govern which incoming messages are persisted
func persist_inbound(s Store, m proto.Message) {
	switch m := m.(type) {
	case *proto.PubAck:
		if m.Header.QosLevel == proto.QosAtMostOnce {
			s.Del(obound_mid2key(m.MessageId))
		}

	case *proto.SubAck:
		if m.Header.QosLevel == proto.QosAtMostOnce {
			s.Del(obound_mid2key(m.MessageId))
		}

	case *proto.Publish:
		if m.Header.QosLevel == proto.QosAtMostOnce {
			s.Del(obound_mid2key(m.MessageId))
		} else if m.Header.QosLevel == proto.QosAtLeastOnce {
			// Received a publish. store it in ibound
			// until puback sent
			s.Put(ibound_mid2key(m.MessageId), m)
		}

	case *proto.Subscribe:
		// sending Subscribe, store in obound until SubAck received
		s.Put(obound_mid2key(m.MessageId), m)

	case *proto.Unsubscribe:
		// until UnsubAck received
		s.Put(obound_mid2key(m.MessageId), m)
	}

}
