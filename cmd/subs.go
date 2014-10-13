package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	mqtt "github.com/funkygao/mhub/broker"
	proto "github.com/funkygao/mqttmsg"
)

var host = flag.String("host", "localhost:1883", "hostname of broker")
var id = flag.String("id", "", "client id")
var user = flag.String("user", "", "username")
var pass = flag.String("pass", "", "password")
var dump = flag.Bool("dump", false, "dump messages?")
var conns = flag.Int("conns", 200, "how many conns")

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sub topic [topic topic...]")
		return
	}
	for i := 0; i < flag.NArg(); i++ {
		for j := 0; j < *conns; j++ {
			go subscribe(flag.Arg(i), j)
		}
	}

	<-make(chan bool)
}

func subscribe(topic string, no int) {
	conn, err := net.Dial("tcp", *host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	cc := mqtt.NewClientConn(conn)
	cc.Dump = *dump
	cc.ClientId = *id

	if err := cc.Connect(*user, *pass); err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected with client id ", cc.ClientId)

	tq := make([]proto.TopicQos, 1)
	tq[0].Topic = topic
	tq[0].Qos = proto.QosAtMostOnce
	cc.Subscribe(tq)

	for m := range cc.Incoming {
		if *dump {
			fmt.Print(m.TopicName, "\t")
			m.Payload.WritePayload(os.Stdout)
			fmt.Println("\tr: ", m.Header.Retain)
		}

	}
}
