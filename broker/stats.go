package broker

import (
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

func (s *stats) messageRecv()      { atomic.AddInt64(&s.recv, 1) }
func (s *stats) messageSend()      { atomic.AddInt64(&s.sent, 1) }
func (s *stats) clientConnect()    { atomic.AddInt64(&s.clients, 1) }
func (s *stats) clientDisconnect() { atomic.AddInt64(&s.clients, -1) }
