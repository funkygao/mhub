package config

import (
	conf "github.com/funkygao/jsconf"
	"time"
)

type RedisConfig struct {
	Server      string
	MaxIdle     int
	IdleTimeout time.Duration
}

func (this *RedisConfig) loadConfig(cf *conf.Conf) {
	this.Server = cf.String("server", "")
	this.MaxIdle = cf.Int("max_idle", 5)
	this.IdleTimeout = cf.Duration("idle_timeout", time.Second*360)
}
