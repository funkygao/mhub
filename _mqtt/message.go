package mqtt

import (
	"fmt"
	log "github.com/cihub/seelog"
	"io"
	"net"
	"sync/atomic"
)

// FixedHeader, VariableHeader, Payload
// byte1
// ==============================
// 7     6 5 4   3   2      1   0
// |         |   |   |      |   |
// [ msg typ ] [dup] [qos lv] [retain]
//
// byte2
// ==============================
// 7                            0
// |                            |
// [    remaining length        ]
//
// all data values are in big-endian order
type FixedHeader struct {
	MessageType uint8
	DupFlag     bool
	QosLevel    uint8
	Retain      bool   // only used on PUBLISH, server should hold on to the message after it has been delivered to the current subscribers
	Length      uint32 // the number of bytes remaining within the current message, including data in the variable header and the payload
}

func (this *FixedHeader) messageTypeStr() string {
	var strArray = []string{
		"reserved",
		"CONNECT",
		"CONNACK",
		"PUBLISH",
		"PUBACK",
		"PUBREC",
		"PUBREL",
		"PUBCOMP",
		"SUBSCRIBE",
		"SUBACK",
		"UNSUBSCRIBE",
		"UNSUBACK",
		"PINGREQ",
		"PINGRESP",
		"DISCONNEC"}
	if this.MessageType > uint8(len(strArray)) {
		return "Undefined MessageType"
	}
	return strArray[this.MessageType]
}

func ReadFixedHeader(conn *net.Conn) *FixedHeader {
	var buf = make([]byte, 2)
	n, _ := io.ReadFull(*conn, buf)
	if n != len(buf) {
		log.Error("read header failed")
		return nil
	}

	byte1 := buf[0]
	header := new(FixedHeader)
	header.MessageType = uint8(byte1 & 0xF0 >> 4)
	header.DupFlag = byte1&0x08 > 0
	header.QosLevel = uint8(byte1 & 0x06 >> 1)
	header.Retain = byte1&0x01 > 0

	byte2 := buf[1]
	header.Length = decodeVarLength(byte2, conn)
	return header
}

func decodeVarLength(cur byte, conn *net.Conn) uint32 {
	length := uint32(0)
	multi := uint32(1)

	for {
		length += multi * uint32(cur&0x7f) // 127
		if cur&0x80 == 0 {                 // continuation?
			break
		}

		buf := make([]byte, 1)
		n, _ := io.ReadFull(*conn, buf)
		if n != 1 {
			panic("failed to read variable length in MQTT header")
		}
		cur = buf[0]
		multi *= 128
	}

	return length
}

func ReadCompleteCommand(conn *net.Conn) (*FixedHeader, []byte) {
	fixed_header := ReadFixedHeader(conn)
	if fixed_header == nil {
		log.Debug("failed to read fixed header")
		return nil, make([]byte, 0)
	}
	length := fixed_header.Length
	buf := make([]byte, length)
	n, _ := io.ReadFull(*conn, buf)
	if uint32(n) != length {
		panic(fmt.Sprintf("failed to read %d bytes specified in fixed header, only %d read", length, n))
	}
	log.Debugf("Complete command(%s) read into buffer", fixed_header.messageTypeStr())

	return fixed_header, buf
}

/*
 This is the type represents a message received from publisher.
 FlyingMessage(message should be delivered to specific subscribers)
 reference MqttMessage
*/
type MqttMessage struct {
	Topic          string
	Payload        string // just a sequence of bytes, up to 256MB
	Qos            uint8
	SenderClientId string
	MessageId      uint16
	InternalId     uint64
	CreatedAt      int64
	Retain         bool // catch up mechanism
}

func (msg *MqttMessage) RedisKey() string {
	return fmt.Sprintf("gossipd.mqtt-msg.%d", msg.InternalId)
}

func (msg *MqttMessage) Store() {
	key := msg.RedisKey()
	G_redis_client.Store(key, msg)
	G_redis_client.Expire(key, 7*24*3600)
}

func CreateMqttMessage(topic, payload, sender_id string,
	qos uint8, message_id uint16,
	created_at int64, retain bool) *MqttMessage {

	msg := new(MqttMessage)
	msg.Topic = topic
	msg.Payload = payload
	msg.Qos = qos
	msg.SenderClientId = sender_id
	msg.MessageId = message_id
	msg.InternalId = GetNextMessageInternalId()
	msg.CreatedAt = created_at
	msg.Retain = retain

	G_messages_lock.Lock()
	G_messages[msg.InternalId] = msg
	G_messages_lock.Unlock()

	msg.Store()

	return msg
}

// CONNECT parse
func parseConnectInfo(buf []byte) *ConnectInfo {
	var info = new(ConnectInfo)
	info.Protocol, buf = parseUTF8(buf)
	info.Version, buf = parseUint8(buf)
	flagByte := buf[0]
	log.Debug("parsing connect flag:", flagByte)
	info.UsernameFlag = (flagByte & 0x80) > 0
	info.PasswordFlag = (flagByte & 0x40) > 0
	info.WillRetain = (flagByte & 0x20) > 0
	info.WillQos = uint8(flagByte & 0x18 >> 3)
	info.WillFlag = (flagByte & 0x04) > 0
	info.CleanSession = (flagByte & 0x02) > 0
	buf = buf[1:]
	info.Keepalive, _ = parseUint16(buf)
	return info
}

func parseUint16(buf []byte) (uint16, []byte) {
	return uint16(buf[0]<<8) + uint16(buf[1]), buf[2:]
}

func parseUint8(buf []byte) (uint8, []byte) {
	return uint8(buf[0]), buf[1:]
}

func parseUTF8(buf []byte) (string, []byte) {
	length, buf := parseUint16(buf)
	str := buf[:length]
	return string(str), buf[length:]
}

type Mqtt struct {
	FixedHeader                                                                   *FixedHeader
	ProtocolName, TopicName, ClientId, WillTopic, WillMessage, Username, Password string
	ProtocolVersion                                                               uint8
	ConnectFlags                                                                  *ConnectFlags
	KeepAliveTimer, MessageId                                                     uint16
	Data                                                                          []byte
	Topics                                                                        []string
	Topics_qos                                                                    []uint8
	ReturnCode                                                                    uint8
}

// CleanSession: discard all state info at conn/disconn
type ConnectFlags struct {
	UsernameFlag, PasswordFlag, WillRetain, WillFlag, CleanSession bool
	WillQos                                                        uint8
}

type ConnectInfo struct {
	Protocol     string // Must be 'MQIsdp' for now
	Version      uint8
	UsernameFlag bool
	PasswordFlag bool
	WillRetain   bool
	WillQos      uint8
	WillFlag     bool
	CleanSession bool
	Keepalive    uint16
}

func GetNextMessageInternalId() uint64 {
	return atomic.AddUint64(&g_next_mqtt_message_internal_id, 1)
}

// This is thread-safe
func GetMqttMessageById(internal_id uint64) *MqttMessage {
	key := fmt.Sprintf("gossipd.mqtt-msg.%d", internal_id)

	msg := new(MqttMessage)
	G_redis_client.Fetch(key, msg)
	return msg
}

/*
 This is the type represents a message should be delivered to
 specific client
*/
type FlyingMessage struct {
	Qos               uint8 // the Qos in effect
	DestClientId      string
	MessageInternalId uint64 // The MqttMessage of interest
	Status            uint8  // The status of this message, like PENDING_PUB(deliver occured
	// when client if offline), PENDING_ACK, etc
	ClientMessageId uint16 // The message id to be used in MQTT packet
}

func CreateFlyingMessage(dest_id string, message_internal_id uint64,
	qos uint8, status uint8, message_id uint16) *FlyingMessage {
	msg := new(FlyingMessage)
	msg.Qos = qos
	msg.DestClientId = dest_id
	msg.MessageInternalId = message_internal_id
	msg.Status = status
	msg.ClientMessageId = message_id
	return msg
}
