package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/funkygao/golib/color"
	mqtt "github.com/funkygao/mhub/broker"
	proto "github.com/funkygao/mqttmsg"
	"net"
	"os"
)

var (
	server string
	topic  string
	user   string

	cc *mqtt.ClientConn
)

func init() {
	flag.StringVar(&server, "server", "192.168.22.73:1883", "broker addr")
	flag.StringVar(&topic, "topic", "world", "topic name")
	flag.StringVar(&user, "user", "anonymous", "chat user name")
	flag.Parse()
}

func main() {
	setupChat()
	cliLoop()
}

func setupChat() {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: ", err)
		os.Exit(1)
	}
	cc = mqtt.NewClientConn(conn, 100)
	cc.KeepAlive = 600
	if err := cc.Connect("", ""); err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Broker connected with client id: ", cc.ClientId)

	cc.Subscribe([]proto.TopicQos{proto.TopicQos{Topic: topic, Qos: proto.QosAtMostOnce}})
	go subLoop()
}

func subLoop() {
	var text string
	for m := range cc.Incoming {
		text = string(m.Payload.(proto.BytesPayload))
		fmt.Printf("[%s] -> %s\n", color.Yellow(m.TopicName), color.Red(text))
	}
}

func cliLoop() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("[%s] Enter text: ", topic)
		text, _ := reader.ReadString('\n')
		text = text[:len(text)-1] // strip EOL
		if text == "" {
			continue
		}

		cc.Publish(&proto.Publish{
			Header:    proto.Header{},
			TopicName: topic,
			Payload:   proto.BytesPayload([]byte(user + ":" + text)),
		})
	}

}
