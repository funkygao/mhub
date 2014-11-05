package broker

import (
	log "github.com/funkygao/log4go"
	"github.com/funkygao/mhub/config"
)

// TODO push if a friend is just online
type Presence struct {
	store *RedisStore
}

func NewPresence(cf config.RedisConfig) *Presence {
	this := new(Presence)
	this.store = NewRedisStore(cf)
	return this
}

func (this *Presence) CheckIn(username string) {
	log.Debug("check in: %s", username)
	this.store.Store(PRESENCE_KEY_PREFIX+username, true)
}

func (this *Presence) CheckOut(username string) {
	log.Debug("check out: %s", username)
	this.store.Del(PRESENCE_KEY_PREFIX + username)
}
