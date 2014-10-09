package broker

import (
	proto "github.com/funkygao/mqttmsg"
)

// A retain holds information necessary to correctly manage retained
// messages.
//
// This needs to hold copies of the proto.Publish, not pointers to
// it, or else we can send out one with the wrong retain flag.
type retain struct {
	m    proto.Publish
	wild wild
}

// A post is a unit of work for the subscription processing workers.
type post struct {
	c *incomingConn
	m *proto.Publish
}

type receipt chan struct{}

// Wait for the receipt to indicate that the job is done.
func (r receipt) wait() {
	// TODO: timeout
	<-r
}

type job struct {
	m proto.Message
	r receipt
}
