package broker

import (
	log "github.com/funkygao/log4go"
	"github.com/funkygao/mhub/config"
	proto "github.com/funkygao/mqttmsg"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

// An incomingConn represents a MQTT connection into a Server.
type incomingConn struct {
	server *Server

	flag *proto.Connect // nil if not CONNECT ok

	alive         bool
	conn          net.Conn
	jobs          chan job
	heartbeatStop chan struct{}
	store         Store
	lastOpTime    int64 // // Last Unix timestamp when recieved message from this conn
}

func (this *incomingConn) String() string {
	if this.flag == nil {
		// CONNECT not sent yet
		return this.conn.RemoteAddr().String()
	}

	return this.flag.ClientId + "@" + this.conn.RemoteAddr().String()
}

// race:
// client disconnect -> inboundLoop close jobs ->
// outboundLoop stops -> outboundLoop close conn, unsubAll, del
func (this *incomingConn) inboundLoop() {
	defer func() {
		this.server.stats.clientDisconnect()
		this.store.Close()

		this.alive = false // to avoid send on closed channel subs.c.submit FIXME
		close(this.jobs)   // will terminate outboundLoop
	}()

	for {
		// FIXME client connected, but idle for long, should kill it
		m, err := proto.DecodeOneMessage(this.conn, nil)
		if err != nil {
			if err != io.EOF && !strings.HasSuffix(err.Error(), errTcpUseOfClosedNetwork) {
				// e,g. read tcp 106.49.97.242:62547: connection reset by peer
				// e,g. read tcp 127.0.0.1:65256: operation timed out
				log.Error("%v: %s", err, this)
			}

			this.server.stats.aborted()

			return
		}

		this.server.stats.messageRecv()
		this.server.stats.addIn(m)
		this.refreshOpTime()

		if this.server.cf.Broker.Echo {
			log.Debug("%s -> %T %+v", this, m, m)
		}

		switch m := m.(type) {
		case *proto.Connect: // TODO close conn if too long no Connect
			rc := this.doConnect(m)

			// close connection if it was a bad connect
			if rc != proto.RetCodeAccepted {
				log.Error("%v: %s", proto.ConnectionErrors[rc], this)
				return
			}

			// connect ok
			log.Debug("new client: %s (c^%v, k^%v)",
				this, m.CleanSession, m.KeepAliveTimer)

		case *proto.Publish:
			this.doPublish(m)

		case *proto.Subscribe:
			this.doSubscribe(m)

		case *proto.Unsubscribe:
			this.doUnsubscribe(m)

		case *proto.PubAck:
			this.doPublishAck(m)

		case *proto.PingReq:
			// broker will never ping client
			this.validateMessage(m)
			this.submit(&proto.PingResp{})

		case *proto.Disconnect:
			log.Debug("%s actively disconnect", this)
			return

		default:
			log.Warn("%s -> unexpected %T", this, m)
			return
		}
	}
}

func (this *incomingConn) outboundLoop() {
	defer func() {
		// close connection on exit in order to cause inboundLoop to exit.
		// only outboundLoop can close conn,
		// otherwise outboundLoop will error: use of closed network connection
		log.Debug("%s conn closed", this)
		close(this.heartbeatStop)
		this.conn.Close()

		this.del()
		this.server.subs.unsubAll(this)
	}()

	var (
		t1      time.Time
		elapsed time.Duration
		totalN  int64
		slowN   int64
	)
	for {
		select {
		case job, on := <-this.jobs:
			if !on {
				// jobs chan was closed by inboundLoop
				return
			}

			if this.server.cf.Broker.Echo {
				log.Debug("%s <- %T %+v", this, job.m, job.m)
			}

			persist_outbound(this.store, job.m)

			t1 = time.Now()
			this.conn.SetWriteDeadline(t1.Add(this.server.cf.Broker.IOTimeout))
			err := job.m.Encode(this.conn)
			elapsed = time.Since(t1)
			if job.r != nil {
				// notifiy the sender that this message is sent
				close(job.r)
			}
			if err != nil {
				// e,g. write tcp 127.0.0.1:59919: connection reset by peer
				// e,g. write tcp 106.49.97.242:4341: broken pipe
				// e,g. write tcp 106.49.97.242:61016: i/o timeout
				// try:
				//     sock.write('foo')
				// except:
				//     pass # connection reset by peer
				// sock.write('bar') # broken pipe
				log.Error("client[%s]: %s, %s", this, err, elapsed)
				return
			}

			totalN++
			if elapsed.Nanoseconds() > this.server.cf.Broker.ClientSlowThreshold.Nanoseconds() {
				slowN++
				log.Warn("Slow client[%s] %d/%d, %s", this, slowN, totalN, elapsed)
			}

			this.server.stats.messageSend()
			this.server.stats.addOut(job.m)
			this.refreshOpTime()

			if _, ok := job.m.(*proto.Disconnect); ok {
				return
			}
		}
	}

}

func (this *incomingConn) refreshOpTime() {
	atomic.StoreInt64(&this.lastOpTime, time.Now().Unix())
}

func (this *incomingConn) heartbeat(keepAliveTimer time.Duration) {
	ticker := time.NewTicker(keepAliveTimer)
	defer func() {
		ticker.Stop()
		log.Debug("%s hearbeat stopped", this)
	}()

	for {
		select {
		case <-ticker.C:
			// 1.5*KeepAliveTimer latency tolerance
			deadline := int64(float64(this.lastOpTime) + keepAliveTimer.Seconds()*1.5)
			overIdle := time.Now().Unix() - deadline
			if overIdle > 0 && this.alive {
				this.submitSync(&proto.Disconnect{}).wait()
				log.Warn("%s over idle %ds, kicked out", this, overIdle)

				this.server.stats.aborted()

				if this.flag != nil && this.flag.WillFlag {
					// TODO broker will publish a message on behalf of the client
				}

				return
			}

		case <-this.heartbeatStop:
			return
		}
	}

}

func (this *incomingConn) connected() bool {
	return this.flag != nil
}

func (this *incomingConn) add() *incomingConn {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	existing, present := clients[this.flag.ClientId]
	if present {
		return existing
	}

	clients[this.flag.ClientId] = this
	return nil
}

// Delete a connection; the conection must be closed by the caller first.
func (this *incomingConn) del() {
	clientsMu.Lock()
	if this.flag != nil {
		delete(clients, this.flag.ClientId)
	}
	clientsMu.Unlock()
}

func (this *incomingConn) submit(m proto.Message) {
	if !this.alive {
		log.Debug("%s submit on dead client: %T %+v", this, m, m)
		return
	}

	if this.server.cf.Broker.BuffOverflowStrategy == config.BufferOverflowBlock {
		this.jobs <- job{m: m}
		return
	}

	// config.BufferOverflowDiscard
	select {
	case this.jobs <- job{m: m}:
	default:
		log.Debug("client[%s]: outbound(%d) full, discard %T", this,
			len(this.jobs), m)
	}
}

// Queue a message, returns a channel that will be readable
// when the message is sent.
func (this *incomingConn) submitSync(m proto.Message) receipt {
	j := job{m: m, r: make(receipt)}
	this.jobs <- j
	return j.r
}
