package broker

import (
	"github.com/funkygao/mhub/config"
	"github.com/funkygao/redigo/redis"
	"time"
)

type redisClient struct {
	pool *redis.Pool
}

func newRedisClient(cf config.RedisConfig) *redisClient {
	this := new(redisClient)
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
