package main

import (
	"fmt"
	"github.com/funkygao/golib/server"
	"github.com/funkygao/mhub/broker"
	"github.com/funkygao/mhub/config"
	"runtime/debug"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
	}()

	server := server.NewServer("mhub")
	server.LoadConfig(option.configFile)
	server.Launch()

	broker := broker.NewServer(config.LoadConfig(server.Conf))
	broker.Start()
	<-broker.Done
}
