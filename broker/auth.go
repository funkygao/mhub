package broker

import (
	proto "github.com/funkygao/mqttmsg"
)

// TODO block user if failed many times
func (this *incomingConn) authenticate(username, passwd string) (ok bool) {
	ok = true
	return
}

func (this *incomingConn) authorized(username string, m proto.Message) (ok bool) {
	// subscribe
	// publish
	// spam control
	ok = true
	return
}
