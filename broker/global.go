package broker

import (
	"errors"
	"math/rand"
	"sync"
)

// all clients in mem, key is client id
var clients map[string]*incomingConn = make(map[string]*incomingConn)
var clientsMu sync.Mutex

// A random number generator ready to make client-id's, if
// they do not provide them to us.
var clientIdRand *rand.Rand

// ConnectionErrors is an array of errors corresponding to the
// Connect return codes specified in the specification.
var ConnectionErrors = [6]error{
	nil, // Connection Accepted (not an error)
	errors.New("Connection Refused: unacceptable protocol version"),
	errors.New("Connection Refused: identifier rejected"),
	errors.New("Connection Refused: server unavailable"),
	errors.New("Connection Refused: bad user name or password"),
	errors.New("Connection Refused: not authorized"),
}
