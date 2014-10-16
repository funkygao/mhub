package main

import (
	"flag"
	"fmt"
	"log"
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

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sub topic [topic topic...]")
		return
	}

	conn, err := net.Dial("tcp", *host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	cc := mqtt.NewClientConn(conn)
	cc.Dump = *dump
	cc.KeepAlive = 60
	cc.ClientId = *id

	tq := make([]proto.TopicQos, flag.NArg())
	for i := 0; i < flag.NArg(); i++ {
		tq[i].Topic = flag.Arg(i)
		tq[i].Qos = proto.QosAtMostOnce
	}

	if err := cc.Connect(*user, *pass); err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected with client id ", cc.ClientId)
	cc.Subscribe(tq)

	for m := range cc.Incoming {
		fmt.Print(m.TopicName, "\t")
		m.Payload.WritePayload(os.Stdout)
		fmt.Println("\tr: ", m.Header.Retain)
	}
}
