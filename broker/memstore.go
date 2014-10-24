package broker

import (
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
	"sync"
)

// MemoryStore implements the store interface to provide a "persistence"
// mechanism wholly stored in memory. This is only useful for
// as long as the client instance exists.
// FIXME use LRU to avoid being killed by bad client
type MemoryStore struct {
	sync.RWMutex
	messages map[string]proto.Message
	opened   bool
}

func NewMemoryStore() *MemoryStore {
	this := &MemoryStore{
		messages: make(map[string]proto.Message),
		opened:   false,
	}
	return this
}

func (this *MemoryStore) Open() {
	this.Lock()
	this.opened = true
	this.Unlock()
}

func (this *MemoryStore) Put(key string, message proto.Message) {
	this.Lock()
	this.messages[key] = message
	this.Unlock()
	log.Debug("put -> key:%s, val:%#v", key, message)
}

func (this *MemoryStore) Get(key string) proto.Message {
	this.RLock()
	defer this.RUnlock()
	log.Debug("get -> key:%s", key)
	return this.messages[key]
}

func (this *MemoryStore) All() []string {
	this.RLock()
	defer this.RUnlock()

	keys := []string{}
	for k, _ := range this.messages {
		keys = append(keys, k)
	}
	log.Debug("all: %+v", keys)
	return keys
}

func (this *MemoryStore) Del(key string) {
	this.Lock()
	delete(this.messages, key)
	this.Unlock()
	log.Debug("del %s", key)
}

func (this *MemoryStore) Close() {
	this.Lock()
	this.opened = false
	this.Unlock()
}

func (this *MemoryStore) Reset() {
	this.Lock()
	this.messages = make(map[string]proto.Message)
	this.Unlock()
	log.Debug("reset")
}
