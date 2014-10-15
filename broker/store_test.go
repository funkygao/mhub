package broker

import (
	"github.com/funkygao/assert"
	"github.com/funkygao/mhub/config"
	proto "github.com/funkygao/mqttmsg"
	"github.com/funkygao/msgpack"
	"github.com/funkygao/redigo/redis"
	"sync"
	"testing"
	"time"
)

func getRedisStore() *RedisStore {
	cf := config.RedisConfig{
		Server:      "localhost:6379",
		MaxIdle:     5,
		IdleTimeout: time.Second * 240,
	}
	return newRedisStore(cf)
}

func TestBasicCRUD(t *testing.T) {
	redis := getRedisStore()
	var val int
	var key = "key"

	val = 98
	redis.Store(key, val)
	redis.Get(key, &val)
	assert.Equal(t, 98, val)
	redis.Del(key)
}

func BenchmarkStorePublishMessage(b *testing.B) {
	b.ReportAllocs()

	redis := getRedisStore()
	var p = proto.Publish{}
	for i := 0; i < b.N; i++ {
		redis.Store("foo", p)
	}
}

func BenchmarkMsgpackMarshalPublishMessage(b *testing.B) {
	b.ReportAllocs()

	var p = proto.Publish{}
	for i := 0; i < b.N; i++ {
		msgpack.Marshal(p)
	}
}

func BenchmarkLockThenUnlock(b *testing.B) {
	var mu sync.Mutex
	for i := 0; i < b.N; i++ {
		mu.Lock()
		mu.Unlock()
	}
}

func BenchmarkRedisPoolGetConn(b *testing.B) {
	b.ReportAllocs()

	cf := config.RedisConfig{
		Server: "localhost:6379",
	}
	pool := &redis.Pool{
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
	for i := 0; i < b.N; i++ {
		pool.Get()
	}
}

func Benchma1rkRawRedisDial(b *testing.B) {
	b.ReportAllocs()

	var server string = ":6379"
	for i := 0; i < b.N; i++ {
		conn, err := redis.Dial("tcp", server)
		if err != nil {
			b.Logf("%v", err)
			continue
		}
		conn.Close()
	}
}
