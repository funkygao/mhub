package broker

import (
	"fmt"
	log "github.com/funkygao/log4go"
	"sync/atomic"
	"time"
)

type stats struct {
	interval time.Duration

	recv       int64
	sent       int64
	clients    int64
	clientsMax int64
	lastmsgs   int64
}

func (this *stats) messageRecv()      { atomic.AddInt64(&this.recv, 1) }
func (this *stats) messageSend()      { atomic.AddInt64(&this.sent, 1) }
func (this *stats) clientConnect()    { atomic.AddInt64(&this.clients, 1) }
func (this *stats) clientDisconnect() { atomic.AddInt64(&this.clients, -1) }

func (this stats) String() string {
	return fmt.Sprintf("{recv:%d, sent:%d, clients:%d, clientsMax:%d, lastmsgs:%d}",
		atomic.LoadInt64(&this.recv), atomic.LoadInt64(&this.sent),
		atomic.LoadInt64(&this.clients), atomic.LoadInt64(&this.clientsMax),
		atomic.LoadInt64(&this.lastmsgs))
}

func (this *stats) start() {
	ticker := time.NewTicker(this.interval)
	defer ticker.Stop()

	for _ = range ticker.C {
		log.Info("stats: %s", *this)

	}
}
