package config

import (
	conf "github.com/funkygao/jsconf"
	log "github.com/funkygao/log4go"
)

type Config struct {
	ListenAddr    string
	TlsListenAddr string
	TlsServerCert string
	TlsServerKey  string
}

func LoadConfig(cf *conf.Conf) *Config {
	this := new(Config)
	this.ListenAddr = cf.String("listen_addr", "")
	this.TlsListenAddr = cf.String("tls_listen_addr", "")
	if this.TlsListenAddr != "" {
		this.TlsServerCert = cf.String("tls_server_cert", "server.crt")
		this.TlsServerKey = cf.String("tls_server_key", "server.key")
	}
	if this.ListenAddr == "" && this.TlsListenAddr == "" {
		panic("Empty listen address")
	}

	log.Debug("config: %+v", *this)
	return this
}
