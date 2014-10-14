package broker

import (
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkChannelClose(b *testing.B) {
	b.ReportAllocs()
	var c chan bool
	for i := 0; i < b.N; i++ {
		c = make(chan bool)
		close(c)
	}
}

func BenchmarkIsWild(b *testing.B) {
	const topic = "system/user/1212121212121/private"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		isWildcard(topic)
	}
	b.SetBytes(int64(len(topic)))
}

func BenchmarkAtomicAdd(b *testing.B) {
	var a int32
	for i := 0; i < b.N; i++ {
		atomic.AddInt32(&a, 10)
	}
}

func BenchmarkSubscribers(b *testing.B) {
	b.ReportAllocs()

	cs := make([]*incomingConn, 0)
	for i := 0; i < 10000; i++ {
		c := &incomingConn{
			alive:         true,
			conn:          nil,
			jobs:          make(chan job, 10),
			heartbeatStop: make(chan struct{}),
			lastOpTime:    time.Now().Unix(),
		}
		cs = append(cs, c)
	}
	c := &incomingConn{
		alive:         true,
		conn:          nil,
		jobs:          make(chan job, 10),
		heartbeatStop: make(chan struct{}),
		lastOpTime:    time.Now().Unix(),
	}
	subs := newSubscriptions(5, 10, nil)
	for _, c = range cs {
		subs.add("#", c)
	}
	for i := 0; i < b.N; i++ {
		subs.subscribers("ha")
	}
}
