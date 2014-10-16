package broker

import (
	"fmt"
	"github.com/funkygao/golib/gofmt"
	"github.com/funkygao/golib/server"
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
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

	topics    int64 // TODO
	replBytes int64
	inBytes   int64
	outBytes  int64
	repl      int64
	recv      int64
	sent      int64
	sessions  int64 // cumulated
	aborts    int64 // cumulated aborted client sessions
	clients   int64
	peers     int64
}

func (this *stats) addIn(m proto.Message)   { atomic.AddInt64(&this.inBytes, int64(m.Bytes())) }
func (this *stats) addOut(m proto.Message)  { atomic.AddInt64(&this.outBytes, int64(m.Bytes())) }
func (this *stats) addRepl(m proto.Message) { atomic.AddInt64(&this.replBytes, int64(m.Bytes())) }
func (this *stats) aborted()                { atomic.AddInt64(&this.aborts, 1) }
func (this *stats) replicated()             { atomic.AddInt64(&this.repl, 1) }
func (this *stats) messageRecv()            { atomic.AddInt64(&this.recv, 1) }
func (this *stats) messageSend()            { atomic.AddInt64(&this.sent, 1) }
func (this *stats) clientConnect() {
	atomic.AddInt64(&this.clients, 1)
	atomic.AddInt64(&this.sessions, 1)
}
func (this *stats) clientDisconnect() { atomic.AddInt64(&this.clients, -1) }
func (this *stats) peerConnect()      { atomic.AddInt64(&this.peers, 1) }
func (this *stats) peerDisconnect()   { atomic.AddInt64(&this.peers, -1) }

func (this *stats) String() string {
	return fmt.Sprintf("{recv:%d, repl:%d/%d, sent:%d/%d, sess:%d, abort:%d}",
		atomic.LoadInt64(&this.recv),
		atomic.LoadInt64(&this.repl),
		atomic.LoadInt64(&this.peers),
		atomic.LoadInt64(&this.sent),
		atomic.LoadInt64(&this.clients),
		atomic.LoadInt64(&this.sessions),
		atomic.LoadInt64(&this.aborts))
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
		startedAt    = time.Now()
		totalPackets int64
		lastPackets  int64
		throughput   int64
		nsInMs       uint64 = 1000 * 1000
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

		totalPackets = atomic.LoadInt64(&this.recv) + atomic.LoadInt64(&this.sent)
		throughput = (totalPackets - lastPackets) / int64(this.interval.Seconds())
		lastPackets = totalPackets

		log.Info("ver:%s, since:%s, go:%d, mem:%s, gc:%dms/%d=%d, heap:%s,%s,%s,%s cpu:{%3.2f%%us, %3.2f%%sy}",
			server.BuildID,
			time.Since(startedAt),
			runtime.NumGoroutine(),
			gofmt.ByteSize(ms.Alloc),
			ms.PauseTotalNs/nsInMs,
			ms.NumGC,
			ms.PauseTotalNs/(nsInMs*uint64(ms.NumGC)),
			gofmt.ByteSize(ms.HeapSys),      // bytes it has asked the operating system for
			gofmt.ByteSize(ms.HeapAlloc),    // bytes currently allocated in the heap
			gofmt.ByteSize(ms.HeapIdle),     // bytes in the heap that are unused
			gofmt.ByteSize(ms.HeapReleased), // bytes returned to the operating system, 5m for scavenger
			userCpuUtil,
			sysCpuUtil)

		log.Info("%d/s, %s {in:%s, out:%s, repl:%s}",
			throughput,
			this,
			gofmt.ByteSize(this.inBytes),
			gofmt.ByteSize(this.outBytes),
			gofmt.ByteSize(this.replBytes))

		log.Info("")
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
