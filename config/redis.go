package config

import (
	conf "github.com/funkygao/jsconf"
)

type RedisConfig struct {
	Server string
}

func (this *RedisConfig) loadConfig(cf *conf.Conf) {
	this.Server = cf.String("server", "")
}
