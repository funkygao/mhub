package broker

import (
	log "github.com/funkygao/log4go"
	"github.com/funkygao/mhub/config"
	"github.com/funkygao/msgpack"
	"github.com/funkygao/redigo/redis"
	"sync"
	"time"
)

type RedisStore struct {
	pool *redis.Pool
	mu   sync.Mutex // redis.Do is not goroutine safe
}

func newRedisStore(cf config.RedisConfig) *RedisStore {
	this := new(RedisStore)
	this.pool = &redis.Pool{
		MaxIdle:     cf.MaxIdle,
		IdleTimeout: cf.IdleTimeout,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cf.Server)
			if err != nil {
				return nil, err
			}

			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return this
}

func (this *RedisStore) Store(key string, val interface{}) {
	this.mu.Lock()
	m, err := msgpack.Marshal(val)
	if err != nil {
		log.Error(err)
	}
	_, err = this.pool.Get().Do("SET", key, m)
	if err != nil {
		log.Error(err)
	}
	this.mu.Unlock()
}

func (this *RedisStore) Get(key string, val interface{}) {
	this.mu.Lock()
	m, err := redis.Bytes(this.pool.Get().Do("GET", key))
	if err != nil {
		log.Error(err)
	}
	err = msgpack.Unmarshal(m, val)
	if err != nil {
		log.Error(err)
	}
	this.mu.Unlock()
}

func (this *RedisStore) Del(key string) {
	this.mu.Lock()
	this.pool.Get().Do("DEL", key)
	this.mu.Unlock()
}

func (this *RedisStore) Expire(key string, sec uint64) {
	this.mu.Lock()
	_, err := this.pool.Get().Do("EXPIRE", key, sec)
	if err != nil {
		log.Error(err)
	}
	this.mu.Unlock()
}
