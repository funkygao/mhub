package broker

import (
	"fmt"
	"github.com/funkygao/golib/gofmt"
	"github.com/funkygao/golib/server"
	log "github.com/funkygao/log4go"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

type stats struct {
	interval time.Duration

	topics   int64 // TODO
	recv     int64
	sent     int64
	sessions int64 // cumulated
	clients  int64
	peers    int64
}

func (this *stats) messageRecv() { atomic.AddInt64(&this.recv, 1) }
func (this *stats) messageSend() { atomic.AddInt64(&this.sent, 1) }
func (this *stats) clientConnect() {
	atomic.AddInt64(&this.clients, 1)
	atomic.AddInt64(&this.sessions, 1)
}
func (this *stats) clientDisconnect() { atomic.AddInt64(&this.clients, -1) }
func (this *stats) peerConnect()      { atomic.AddInt64(&this.peers, 1) }
func (this *stats) peerDisconnect()   { atomic.AddInt64(&this.peers, -1) }

func (this *stats) String() string {
	return fmt.Sprintf("{topic:%d, recv:%d, sent:%d, sess:%d, client:%d, peer:%d}",
		atomic.LoadInt64(&this.topics),
		atomic.LoadInt64(&this.recv),
		atomic.LoadInt64(&this.sent),
		atomic.LoadInt64(&this.sessions),
		atomic.LoadInt64(&this.clients),
		atomic.LoadInt64(&this.peers))
}

// current simultaneous client conns
func (this *stats) Clients() int {
	return int(atomic.LoadInt64(&this.clients))
}

func (this *stats) start() {
	ticker := time.NewTicker(this.interval)
	defer ticker.Stop()

	var (
		ms           = new(runtime.MemStats)
		rusage       = &syscall.Rusage{}
		lastUserTime int64
		lastSysTime  int64
		userTime     int64
		sysTime      int64
		userCpuUtil  float64
		sysCpuUtil   float64
	)
	for _ = range ticker.C {
		runtime.ReadMemStats(ms)

		syscall.Getrusage(syscall.RUSAGE_SELF, rusage)
		syscall.Getrusage(syscall.RUSAGE_SELF, rusage)
		userTime = rusage.Utime.Sec*1000000000 + int64(rusage.Utime.Usec)
		sysTime = rusage.Stime.Sec*1000000000 + int64(rusage.Stime.Usec)
		userCpuUtil = float64(userTime-lastUserTime) * 100 / float64(this.interval)
		sysCpuUtil = float64(sysTime-lastSysTime) * 100 / float64(this.interval)

		lastUserTime = userTime
		lastSysTime = sysTime

		log.Info("%s, ver:%s, goroutine:%d, mem:%s, cpu:{%3.2f%%us, %3.2f%%sy}",
			this,
			server.BuildID,
			runtime.NumGoroutine(),
			gofmt.ByteSize(ms.Alloc),
			userCpuUtil,
			sysCpuUtil)
	}
}
