package main

import (
	"flag"
	"github.com/funkygao/golib/server"
)

var (
	option struct {
		configFile  string
		showVersion bool
	}
)

func init() {
	flag.StringVar(&option.configFile, "conf", "etc/mqttd.cf", "config file")
	flag.BoolVar(&option.showVersion, "version", false, "show version and exit")

	flag.Parse()

	if option.showVersion {
		server.ShowVersionAndExit()
	}
}
