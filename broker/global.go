package broker

import (
	"sync"
)

// all clients in mem, key is client id
var clients map[string]*incomingConn = make(map[string]*incomingConn, initClientNum)
var clientsMu sync.Mutex

// TODO discard client store after disconnect for some time
var clientStores map[string]Store = make(map[string]Store, initClientNum) // key is clientId
var clientStoresMu sync.Mutex
