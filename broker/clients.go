package broker

import (
	"sync"
)

type clients1 struct {
	conns map[string]*incomingConn
	mu    sync.Mutex
}

func (this *clients1) add(c *incomingConn) {
	this.mu.Lock()
	this.conns[c.flag.ClientId] = c
	this.mu.Unlock()
}
