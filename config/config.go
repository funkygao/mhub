package config

import (
	conf "github.com/funkygao/jsconf"
	log "github.com/funkygao/log4go"
)

type Config struct {
	ListenAddr string
}

func LoadConfig(cf *conf.Conf) *Config {
	this := new(Config)
	this.ListenAddr = cf.String("listen_addr", "")
	if this.ListenAddr == "" {
		panic("Empty listen address")
	}

	log.Debug("config: %+v", *this)
	return this
}
