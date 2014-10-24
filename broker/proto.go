package broker

import (
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
	"time"
)

// TODO
func (this *incomingConn) validateMessage(m proto.Message) {
	// must CONNECT before other methods
}

// TODO
func (this *incomingConn) nextInternalMsgId() {
	//this.server.cf.Peers.SelfId
}

func (this *incomingConn) doConnect(m *proto.Connect) (rc proto.ReturnCode) {
	rc = proto.RetCodeAccepted // default is ok

	// validate protocol name and version
	if m.ProtocolName != protocolName ||
		m.ProtocolVersion != protocolVersion {
		log.Error("invalid connection[%s] protocol %s, version %d",
			this, m.ProtocolName, m.ProtocolVersion)
		rc = proto.RetCodeUnacceptableProtocolVersion
	}

	// validate client id length
	if len(m.ClientId) < 1 || len(m.ClientId) > maxClientIdLength {
		rc = proto.RetCodeIdentifierRejected
	}
	this.flag = m // connection flag

	// authentication
	if !this.server.cf.Broker.AllowAnonymousConnect &&
		(!m.UsernameFlag || m.Username == "" ||
			!m.PasswordFlag || m.Password == "") {
		rc = proto.RetCodeNotAuthorized
	} else if m.UsernameFlag && !this.authenticate(m.Username, m.Password) {
		rc = proto.RetCodeBadUsernameOrPassword
	}

	// validate clientId should be udid TODO

	if this.server.cf.Broker.MaxConnections > 0 &&
		this.server.stats.Clients() > this.server.cf.Broker.MaxConnections {
		rc = proto.RetCodeServerUnavailable
	}

	// Disconnect existing connections.
	if existing := this.add(); existing != nil {
		log.Warn("found dup client: %s", this)

		// force disconnect existing client
		existing.submitSync(&proto.Disconnect{}).wait()
		existing.del()
	}
	this.add()

	if m.KeepAliveTimer > 0 {
		go this.heartbeat(time.Duration(m.KeepAliveTimer) * time.Second)
	}

	this.store = getClientStore(m.ClientId)

	// TODO: Last will
	// The will option allows clients to prepare for the worst.
	if !m.CleanSession {
		// restore client's subscriptions TODO
		// deliver flying messages FXIME
		for _, key := range this.store.All() {
			if key[0] == 'i' {
				// ibound
			} else {
				// obound
			}
			switch msg := this.store.Get(key).(type) {
			case *proto.Publish:
				if msg.Header.QosLevel == proto.QosAtLeastOnce {
					this.submit(msg)
				}

			case *proto.PubAck:
				if msg.Header.QosLevel == proto.QosAtLeastOnce {
					this.submit(msg)
				}
			}

		}
	} else {
		this.store.Reset()
	}

	this.submit(&proto.ConnAck{ReturnCode: rc})

	return
}

func (this *incomingConn) doPublish(m *proto.Publish) {
	this.validateMessage(m)

	// TODO assert m.TopicName is not wildcard
	persist_inbound(this.store, m)

	// replicate message to all subscribers of this topic
	this.server.subs.submit(m)

	// replication to peers
	if isGlobalTopic(m.TopicName) {
		this.server.peers.submit(m)
	}

	// for QoS 0, we need do nothing
	if m.Header.QosLevel == proto.QosAtLeastOnce { // QoS 1
		if m.MessageId == 0 {
			log.Error("client[%s] invalid message id", this)
		}

		this.submit(&proto.PubAck{MessageId: m.MessageId})
	}

	// retry-until-acknowledged

	// if a PUBLISH not authorized, MQTT has no way of telling client about this
	// it must always make a positive acknowledgement according to QoS

	if m.Retain {

	}
}

func (this *incomingConn) doPublishAck(m *proto.PubAck) {
	this.validateMessage(m)

	// get flying messages for this client
	// if not found, ignore this PubAck
	// if found, mark this flying message
}

func (this *incomingConn) doSubscribe(m *proto.Subscribe) {
	this.validateMessage(m)

	// The SUBSCRIBE message also specifies the QoS level at which the subscriber wants to receive published messages.

	suback := &proto.SubAck{
		MessageId: m.MessageId,
		TopicsQos: make([]proto.QosLevel, len(m.Topics)),
	}
	for i, tq := range m.Topics {
		// TODO: Handle varying QoS correctly
		this.server.subs.add(tq.Topic, this)

		suback.TopicsQos[i] = proto.QosAtMostOnce
	}
	this.submit(suback)

	// A server may start sending PUBLISH messages due to the subscription before the client receives the SUBACK message.

	// Note that if a server implementation does not authorize a SUBSCRIBE request to be made by a client, it has no way of informing that client. It must therefore make a positive acknowledgement with a SUBACK, and the client will not be informed that it was not authorized to subscribe.

	// Process retained messages
	for _, tq := range m.Topics {
		this.server.subs.sendRetain(tq.Topic, this)
	}
}

func (this *incomingConn) doUnsubscribe(m *proto.Unsubscribe) {
	this.validateMessage(m)

	for _, t := range m.Topics {
		this.server.subs.unsub(t, this)
	}

	this.submit(&proto.UnsubAck{MessageId: m.MessageId})
}
