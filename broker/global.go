package broker

import (
	"sync"
)

// all clients in mem, key is client id
var clients map[string]*incomingConn = make(map[string]*incomingConn, 10000)
var clientsMu sync.Mutex
