package main

import (
	"flag"
	"github.com/funkygao/golib/server"
)

var (
	option struct {
		configFile   string
		showVersion  bool
		crashLogFile string
		logFile      string
		logLevel     string
		cpuProf      bool
		memProf      bool
		blockProf    bool
	}
)

func init() {
	flag.StringVar(&option.configFile, "conf", "etc/mhub.cf", "config file")
	flag.BoolVar(&option.showVersion, "version", false, "show version and exit")
	flag.StringVar(&option.crashLogFile, "crashlog", "panic.dump", "crash log")
	flag.StringVar(&option.logFile, "log", "stdout", "log file")
	flag.StringVar(&option.logLevel, "level", "debug", "log level")
	flag.BoolVar(&option.cpuProf, "cpuprof", false, "enable cpu profiler")
	flag.BoolVar(&option.memProf, "memprof", false, "enable mem profiler")
	flag.BoolVar(&option.blockProf, "blockprof", false, "enable block profiler")

	flag.Parse()

	if option.showVersion {
		server.ShowVersionAndExit()
	}

	server.SetupLogging(option.logFile, option.logLevel, option.crashLogFile)
}
