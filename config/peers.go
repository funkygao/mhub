package config

import (
	conf "github.com/funkygao/jsconf"
	"time"
)

type PeersConfig struct {
	ListenAddr string
	TcpNoDelay bool
	Keepalive  bool
	IoTimeout  time.Duration
	Echo       bool
}

func (this *PeersConfig) loadConfig(cf *conf.Conf) {
	this.ListenAddr = cf.String("listen_addr", ":9090")
	this.Keepalive = cf.Bool("keepalive", true)
	this.TcpNoDelay = cf.Bool("tcp_nodelay", true)
	this.IoTimeout = cf.Duration("io_timeout", time.Second*10)
	this.Echo = cf.Bool("echo", false)
}
