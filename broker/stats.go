package broker

import (
	"fmt"
	"github.com/funkygao/golib/gofmt"
	"github.com/funkygao/golib/server"
	log "github.com/funkygao/log4go"
	"github.com/gorilla/mux"
	"net/http"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

type stats struct {
	interval        time.Duration
	statsListenAddr string
	profListenAddr  string

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
	this.launchHttpServ()
	ticker := time.NewTicker(this.interval)
	defer func() {
		ticker.Stop()
		this.stopHttpServ()
	}()

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

func (this *stats) launchHttpServ() {
	if this.statsListenAddr == "" {
		return
	}

	server.LaunchHttpServ(this.statsListenAddr, this.profListenAddr)
	server.RegisterHttpApi("/s/{cmd}",
		func(w http.ResponseWriter, req *http.Request,
			params map[string]interface{}) (interface{}, error) {
			return this.handleHttpQuery(w, req, params)
		}).Methods("GET")
}

func (this *stats) stopHttpServ() {
	server.StopHttpServ()
}

func (this *stats) handleHttpQuery(w http.ResponseWriter, req *http.Request,
	params map[string]interface{}) (interface{}, error) {
	var (
		vars   = mux.Vars(req)
		cmd    = vars["cmd"]
		output = make(map[string]interface{})
	)

	switch cmd {
	case "ping":
		output["status"] = "ok"

	case "ver":
		output["ver"] = server.BuildID

	case "trace":
		stack := make([]byte, 1<<20)
		stackSize := runtime.Stack(stack, true)
		output["callstack"] = string(stack[:stackSize])

	case "sys":
		output["goroutines"] = runtime.NumGoroutine()

		memStats := new(runtime.MemStats)
		runtime.ReadMemStats(memStats)
		output["memory"] = *memStats

		rusage := syscall.Rusage{}
		syscall.Getrusage(0, &rusage)
		output["rusage"] = rusage

	case "stat":
		output["stats"] = this.String()

	}

	output["links"] = []string{
		"/s/ping",
		"/s/ver",
		"s/sys",
		"/s/stat",
		"/s/trace",
	}
	if this.profListenAddr != "" {
		output["pprof"] = "http://" + this.profListenAddr + "/debug/pprof/"
	}

	return output, nil
}
