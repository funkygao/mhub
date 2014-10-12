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

	StatsInterval          time.Duration
	MaxConnections         int // max concurrent client conns
	IOTimeout              time.Duration
	Echo                   bool
	StatsHttpListenAddr    string
	ProfHttpListenAddr     string
	AllowAnonymousConnect  bool
	ClientOutboundQueueLen int
	SubscriptionsWorkers   int
	SubscriptionsQueueLen  int
}

func (this *BrokerConfig) loadConfig(cf *conf.Conf) {
	this.ListenAddr = cf.String("listen_addr", "")
	this.TlsListenAddr = cf.String("tls_listen_addr", "")
	if this.TlsListenAddr != "" {
		this.TlsServerCert = cf.String("tls_server_cert", "server.crt")
		this.TlsServerKey = cf.String("tls_server_key", "server.key")
	}
	this.StatsHttpListenAddr = cf.String("stats_http_listen_addr", "")
	this.StatsInterval = cf.Duration("stats_interval", 10*time.Minute)
	this.ProfHttpListenAddr = cf.String("prof_http_listen_addr", "")
	this.Echo = cf.Bool("echo", false)
	this.SubscriptionsWorkers = cf.Int("subscriptions_workers", 10)
	this.SubscriptionsQueueLen = cf.Int("subscriptions_queue_len", 500)
	this.AllowAnonymousConnect = cf.Bool("allow_anonymous", false)
	this.MaxConnections = cf.Int("max_connections", 50000)
	this.IOTimeout = cf.Duration("io_timeout", time.Second*5)
	this.ClientOutboundQueueLen = cf.Int("client_outbound_queue_len", 100)

	// validation
	if this.ListenAddr == "" && this.TlsListenAddr == "" {
		panic("Empty listen address and tls listen address")
	}
}
