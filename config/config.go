// Config is shared across pkgs
package config

import (
	conf "github.com/funkygao/jsconf"
	log "github.com/funkygao/log4go"
	"time"
)

type Config struct {
	ListenAddr    string
	TlsListenAddr string
	TlsServerCert string
	TlsServerKey  string

	StatsInterval time.Duration
	Echo          bool

	BroadcastWorkers int
}

func LoadConfig(cf *conf.Conf) *Config {
	this := new(Config)
	this.ListenAddr = cf.String("listen_addr", "")
	this.TlsListenAddr = cf.String("tls_listen_addr", "")
	if this.TlsListenAddr != "" {
		this.TlsServerCert = cf.String("tls_server_cert", "server.crt")
		this.TlsServerKey = cf.String("tls_server_key", "server.key")
	}
	this.StatsInterval = cf.Duration("stats_interval", 10*time.Minute)
	this.Echo = cf.Bool("echo", true)
	this.BroadcastWorkers = cf.Int("broadcast_workers", 10)

	// validation
	if this.ListenAddr == "" && this.TlsListenAddr == "" {
		panic("Empty listen address and tls listen address")
	}

	log.Debug("config: %+v", *this)
	return this
}
