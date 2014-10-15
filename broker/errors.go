package broker

import (
	"errors"
)

var (
	errEndpointDupJoin       = errors.New("endpoint dup join")
	errTcpUseOfClosedNetwork = "use of closed network connection"
)
