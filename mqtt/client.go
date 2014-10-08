/* Client representation*/

package mqtt

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	ClientId      string
	Conn          *net.Conn
	WriteLock     *sync.Mutex
	LastTime      int64 // Last Unix timestamp when recieved message from this client
	Shuttingdown  chan uint8
	Subscriptions map[string]uint8
	Mqtt          *Mqtt
	Disconnected  bool
}

func (cr *Client) UpdateLastTime() {
	atomic.StoreInt64(&cr.LastTime, time.Now().Unix())
}

func CreateClient(client_id string, conn *net.Conn, mqtt *Mqtt) *Client {
	rep := new(Client)
	rep.ClientId = client_id
	rep.Conn = conn
	rep.WriteLock = new(sync.Mutex)
	rep.Mqtt = mqtt
	rep.LastTime = time.Now().Unix()
	rep.Shuttingdown = make(chan uint8, 1)
	rep.Subscriptions = make(map[string]uint8)
	rep.Disconnected = false
	return rep
}

func NextOutMessageIdForClient(client_id string) uint16 {
	g_next_client_id_lock.Lock()
	defer g_next_client_id_lock.Unlock()

	next_id, found := NextClientMessageId[client_id]
	if !found {
		NextClientMessageId[client_id] = 1
		return 0
	}
	NextClientMessageId[client_id] = next_id + 1
	return next_id
}
