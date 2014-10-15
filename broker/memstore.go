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

// NewMemoryStore returns a pointer to a new instance of
// MemoryStore, the instance is not initialized and ready to
// use until Open() has been called on it.
func NewMemoryStore() *MemoryStore {
	this := &MemoryStore{
		messages: make(map[string]proto.Message),
		opened:   false,
	}
	return this
}

func (this *MemoryStore) Open() {
	this.opened = true
}

func (this *MemoryStore) Put(key string, message proto.Message) {
	this.Lock()
	defer this.Unlock()
	this.messages[key] = message
}

func (this *MemoryStore) Get(key string) proto.Message {
	this.RLock()
	defer this.RUnlock()

	return this.messages[key]
}

// All returns a slice of strings containing all the keys currently
// in the MemoryStore.
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
	defer this.Unlock()

	delete(this.messages, key)
}

// Close will disallow modifications to the state of the store.
func (this *MemoryStore) Close() {
	this.opened = false
}

// Reset eliminates all persisted message data in the store.
func (this *MemoryStore) Reset() {
	this.Lock()
	defer this.Unlock()
	this.messages = make(map[string]proto.Message)
}
