package main

import (
	"fmt"
	"github.com/funkygao/golib/server"
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
}
