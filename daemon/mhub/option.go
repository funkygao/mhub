package main

import (
	"flag"
	"github.com/funkygao/golib/server"
)

var (
	option struct {
		configFile  string
		showVersion bool
		logFile     string
		logLevel    string
	}
)

func init() {
	flag.StringVar(&option.configFile, "conf", "etc/mhub.cf", "config file")
	flag.BoolVar(&option.showVersion, "version", false, "show version and exit")
	flag.StringVar(&option.logFile, "log", "stdout", "log file")
	flag.StringVar(&option.logLevel, "level", "debug", "log level")

	flag.Parse()

	if option.showVersion {
		server.ShowVersionAndExit()
	}

	server.SetupLogging(option.logFile, option.logLevel)
}
