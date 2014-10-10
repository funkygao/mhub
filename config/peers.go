package config

import (
	conf "github.com/funkygao/jsconf"
)

type PeersConfig struct {
	ListenAddr string
}

func (this *PeersConfig) loadConfig(cf *conf.Conf) {
	this.ListenAddr = cf.String("listen_addr", ":9090")
}
