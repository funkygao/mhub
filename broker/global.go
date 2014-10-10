package broker

import (
	"errors"
	"math/rand"
	"sync"
)

var errUseClosedConn = errors.New("use of closed network connection")

// all clients in mem, key is client id
var clients map[string]*incomingConn = make(map[string]*incomingConn)
var clientsMu sync.Mutex

// A random number generator ready to make client-id's, if
// they do not provide them to us.
var clientIdRand *rand.Rand
