package config

import (
	conf "github.com/funkygao/jsconf"
	"time"
)

type PeersConfig struct {
	SelfId     uint16
	ListenAddr string
	TcpNoDelay bool
	Keepalive  bool
	IoTimeout  time.Duration
	Echo       bool
	QueueLen   int
}

func (this *PeersConfig) loadConfig(cf *conf.Conf) {
	this.SelfId = uint16(cf.Int("self_id", 1))
	this.ListenAddr = cf.String("listen_addr", ":9090")
	this.Keepalive = cf.Bool("keepalive", true)
	this.TcpNoDelay = cf.Bool("tcp_nodelay", true)
	this.IoTimeout = cf.Duration("io_timeout", time.Second*10)
	this.Echo = cf.Bool("echo", false)
	this.QueueLen = cf.Int("queue_len", 1000) // FIXME
}
