package broker

import (
	crand "crypto/rand"
	"fmt"
	proto "github.com/funkygao/mqttmsg"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

// A random number generator ready to make client-id's, if
// they do not provide them to us.
var clientIdRand *rand.Rand

func init() {
	var seed int64
	var sb [4]byte
	crand.Read(sb[:])
	seed = int64(time.Now().Nanosecond())<<32 |
		int64(sb[0])<<24 | int64(sb[1])<<16 |
		int64(sb[2])<<8 | int64(sb[3])
	clientIdRand = rand.New(rand.NewSource(seed))
}

type ClientConn struct {
	conn net.Conn

	ClientId  string
	KeepAlive uint16
	Dump      bool
	Incoming  chan *proto.Publish // subscribed topics downstream channel

	persist    Store
	lastOpTime time.Time
	out        chan job
	doneChan   chan struct{}

	puback  chan *proto.PubAck
	connack chan *proto.ConnAck
	suback  chan *proto.SubAck
}

func NewClientConn(c net.Conn, queueLength int) *ClientConn {
	c.(*net.TCPConn).SetNoDelay(true)
	this := &ClientConn{
		conn:     c,
		persist:  NewMemoryStore(),
		out:      make(chan job, queueLength), // TODO configurable queue len
		Incoming: make(chan *proto.Publish, queueLength),
		doneChan: make(chan struct{}),
		connack:  make(chan *proto.ConnAck),
		suback:   make(chan *proto.SubAck),
		puback:   make(chan *proto.PubAck),
	}
	this.persist.Open()
	go this.inboundLoop()
	go this.outboundLoop()
	return this
}

// timeout is caller's job
func (this *ClientConn) Connect(user, pass string) error {
	if this.ClientId == "" {
		this.ClientId = fmt.Sprint(clientIdRand.Int63())
	}

	req := &proto.Connect{
		ProtocolName:    protocolName,
		ProtocolVersion: protocolVersion,
		ClientId:        this.ClientId,
		CleanSession:    true, // FIXME
		KeepAliveTimer:  this.KeepAlive,
	}
	if this.KeepAlive > 0 {
		go this.runKeepAlive()
	}
	if user != "" {
		req.UsernameFlag = true
		req.PasswordFlag = true
		req.Username = user
		req.Password = pass
	}

	this.sync(req)
	ack := <-this.connack
	return proto.ConnectionErrors[ack.ReturnCode]
}

func (this *ClientConn) Disconnect() {
	this.sync(&proto.Disconnect{})
	<-this.doneChan
}

func (this *ClientConn) Subscribe(topics []proto.TopicQos) *proto.SubAck {
	this.sync(&proto.Subscribe{
		Header:    proto.NewHeader(dupFalse, proto.QosAtLeastOnce, retainFalse),
		MessageId: 0,
		Topics:    topics,
	})
	return <-this.suback
}

func (this *ClientConn) Unsubscribe(m *proto.Unsubscribe) {
	this.out <- job{m: m}
}

func (this *ClientConn) Publish(m *proto.Publish) {
	this.out <- job{m: m}

	switch m.QosLevel {
	case proto.QosAtMostOnce:
		return

	case proto.QosAtLeastOnce:
		<-this.puback

	case proto.QosExactlyOnce:
		panic("not supported QoS")
	}
}

// sync sends a message and blocks until it was actually sent.
func (this *ClientConn) sync(m proto.Message) {
	j := job{m: m, r: make(receipt)}
	this.out <- j
	<-j.r
}

func (this *ClientConn) refreshOpTime() {
	this.lastOpTime = time.Now()
}

func (this *ClientConn) runKeepAlive() {
	ticker := time.NewTicker(time.Duration(this.KeepAlive) * time.Second)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			this.out <- job{m: &proto.PingReq{}}

		case <-this.doneChan:
			return
		}
	}
}

func (this *ClientConn) inboundLoop() {
	defer func() {
		close(this.out) // will terminate outboundLoop

		// Cause any goroutines waiting on messages to arrive to exit.
		close(this.Incoming)

		close(this.connack)
		close(this.suback)
		close(this.puback)
	}()

	for {
		m, err := proto.DecodeOneMessage(this.conn, nil)
		if err != nil {
			if err == io.EOF {
				return
			}

			log.Println(err)
			return
		}

		if this.Dump {
			log.Printf("<- %T", m)
		}

		switch m := m.(type) {
		case *proto.Publish:
			this.Incoming <- m

		case *proto.PubAck:
			this.puback <- m

		case *proto.ConnAck:
			this.connack <- m

		case *proto.SubAck:
			this.suback <- m

		case *proto.Disconnect:
			return

		case *proto.PingResp:
			// ignore this
			continue

		default:
			log.Printf("%s -> unexpected %T", this, m)
		}
	}
}

func (this *ClientConn) outboundLoop() {
	defer func() {
		this.conn.Close() // inbouldLoop will get EOF

		close(this.doneChan)
	}()

	for job := range this.out {
		if this.Dump {
			log.Printf("-> %T", job.m)
		}

		// TODO: write timeout
		err := job.m.Encode(this.conn)
		if job.r != nil {
			close(job.r)
		}

		if err != nil {
			log.Println(err)
			return
		}

		if _, ok := job.m.(*proto.Disconnect); ok {
			return
		}
	}
}
