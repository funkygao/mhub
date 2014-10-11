// Config is shared across pkgs
package config

import (
	conf "github.com/funkygao/jsconf"
	log "github.com/funkygao/log4go"
)

type Config struct {
	Broker BrokerConfig
	Peers  PeersConfig
}

func LoadConfig(cf *conf.Conf) *Config {
	this := new(Config)
	section, err := cf.Section("broker")
	if err != nil {
		panic(err)
	}
	this.Broker = BrokerConfig{}
	this.Broker.loadConfig(section)

	section, err = cf.Section("peers")
	if err == nil {
		this.Peers = PeersConfig{}
		this.Peers.loadConfig(section)
	}

	log.Info("Config: %+v", *this)
	return this
}
