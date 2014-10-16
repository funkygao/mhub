// create a lot of garbage
// once a second create 5-10MB byte array keeps a pool of 20 of the buffers and discard others
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

	var m runtime.MemStats
	makes := 0
	for {
		b := makeBuffer()
		makes += 1
		i := rand.Intn(len(pool))
		pool[i] = b

		time.Sleep(time.Second)

		bytes := 0

		for i := 0; i < len(pool); i++ {
			if pool[i] != nil {
				bytes += len(pool[i])
			}
		}

		runtime.ReadMemStats(&m)
		fmt.Printf("%10s,%10s,%10s,%10s,%10s,%d\n",
			gofmt.ByteSize(m.HeapSys),
			gofmt.ByteSize(bytes),
			gofmt.ByteSize(m.HeapAlloc),
			gofmt.ByteSize(m.HeapIdle),
			gofmt.ByteSize(m.HeapReleased),
			makes)
	}
}
