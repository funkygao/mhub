package main

import (
	"fmt"
	"github.com/funkygao/golib/gofmt"
	"math/rand"
	"runtime"
	"time"
)

func makeBuffer() []byte {
	return make([]byte, rand.Intn(5000000)+5000000)
}

func main() {
	pool := make([][]byte, 20)

	buffer := make(chan []byte, 5)

	var m runtime.MemStats
	makes := 0
	for {
		var b []byte
		select {
		case b = <-buffer:
		default:
			makes += 1
			b = makeBuffer()
		}

		i := rand.Intn(len(pool))
		if pool[i] != nil {
			select {
			case buffer <- pool[i]:
				pool[i] = nil
			default:
			}
		}

		pool[i] = b

		time.Sleep(time.Second)

		bytes := 0
		for i := 0; i < len(pool); i++ {
			if pool[i] != nil {
				bytes += len(pool[i])
			}
		}

		runtime.ReadMemStats(&m)
		fmt.Printf("%10s,%10s,%10s,%10s,%10s,%10d,%12d,%d\n",
			gofmt.ByteSize(m.HeapSys), // bytes it has asked the operating system for
			gofmt.ByteSize(bytes),
			gofmt.ByteSize(m.HeapAlloc),    // bytes currently allocated in the heap
			gofmt.ByteSize(m.HeapIdle),     // bytes in the heap that are unused
			gofmt.ByteSize(m.HeapReleased), // bytes returned to the operating system, 5m for scavenger
			m.NumGC,
			m.PauseTotalNs,
			makes)
	}
}
