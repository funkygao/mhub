package config

import (
	conf "github.com/funkygao/jsconf"
	"time"
)

type BrokerConfig struct {
	ListenAddr    string
	TlsListenAddr string
	TlsServerCert string
	TlsServerKey  string

	StatsInterval  time.Duration
	MaxConnections int // max concurrent client conns
	Echo           bool

	SubscriptionsWorkers  int
	AllowAnonymousConnect bool
}

func (this *BrokerConfig) loadConfig(cf *conf.Conf) {
	this.ListenAddr = cf.String("listen_addr", "")
	this.TlsListenAddr = cf.String("tls_listen_addr", "")
	if this.TlsListenAddr != "" {
		this.TlsServerCert = cf.String("tls_server_cert", "server.crt")
		this.TlsServerKey = cf.String("tls_server_key", "server.key")
	}
	this.StatsInterval = cf.Duration("stats_interval", 10*time.Minute)
	this.Echo = cf.Bool("echo", false)
	this.SubscriptionsWorkers = cf.Int("subscriptions_workers", 10)
	this.AllowAnonymousConnect = cf.Bool("allow_anonymous", false)
	this.MaxConnections = cf.Int("max_connections", 50000)

	// validation
	if this.ListenAddr == "" && this.TlsListenAddr == "" {
		panic("Empty listen address and tls listen address")
	}
}
