package broker

import (
	proto "github.com/funkygao/mqttmsg"
	"sync"
)

// MemoryStore implements the store interface to provide a "persistence"
// mechanism wholly stored in memory. This is only useful for
// as long as the client instance exists.
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
}

func (this *MemoryStore) Get(key string) proto.Message {
	this.RLock()
	defer this.RUnlock()

	return this.messages[key]
}

func (this *MemoryStore) All() []string {
	this.RLock()
	defer this.RUnlock()

	keys := []string{}
	for k, _ := range this.messages {
		keys = append(keys, k)
	}
	return keys
}

func (this *MemoryStore) Del(key string) {
	this.Lock()
	delete(this.messages, key)
	this.Unlock()
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
}
