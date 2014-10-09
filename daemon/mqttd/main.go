package main

import (
	"fmt"
	"github.com/funkygao/golib/server"
	"github.com/funkygao/gomqtt/broker"
	"github.com/funkygao/gomqtt/config"
	"github.com/funkygao/gomqtt/node"
	"runtime/debug"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
	}()

	server := server.NewServer("mqttd")
	server.LoadConfig(option.configFile)
	server.Launch()

	node := node.New(config.LoadConfig(server.Conf))
	node.ServeForever()

	broker := broker.NewServer(config.LoadConfig(server.Conf))
	broker.Start()
	<-broker.Done
}
