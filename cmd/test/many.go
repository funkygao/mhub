/*
Simulates a large number of clients who send a low transaction rate.
The goal is to eventually use this to achieve 1 million (and more?) concurrent, active
MQTT sessions in one server.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	mqtt "github.com/funkygao/mhub/broker"
	proto "github.com/funkygao/mqttmsg"
)

var broadcast = flag.Bool("broadcast", false, "each message will be broadcast to each client")
var conns = flag.Int("conns", 10, "how many conns (0 means infinite)")
var host = flag.String("host", "localhost:1883", "hostname of broker")
var user = flag.String("user", "", "username")
var pass = flag.String("pass", "", "password")
var dump = flag.Bool("dump", false, "dump messages?")
var wait = flag.Int("wait", 20, "ms to wait between client connects")
var pace = flag.Int("pace", 10, "send a message on average once every pace seconds")
var qos = flag.Int("qos", 0, "QoS")
var keepalive = flag.Int("keepalive", 60, "keepalive interval in seconds")

var payload proto.Payload
var topic string

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	if flag.NArg() != 2 {
		topic = "many"
		payload = proto.BytesPayload([]byte(`{"payload":{"330":{"uid":53,"march_id":330,"city_id":53,"opp_uid":0,"world_id":1,"type":"encamp","start_x":72,"start_y":64,"end_x":80,"end_y":78,"start_time":1412999095,"end_time":1412999111,"speed":1,"state":"marching","alliance_id":0}}`))
	} else {
		topic = flag.Arg(0)
		payload = proto.BytesPayload([]byte(flag.Arg(1)))
	}

	if *conns == 0 {
		*conns = -1
	}

	i := 0
	for {
		go client(i)
		i++

		*conns--
		if *conns == 0 {
			break
		}
		time.Sleep(time.Duration(*wait) * time.Millisecond)
	}

	// sleep forever
	<-make(chan struct{})
}

func client(i int) {
	log.Printf("starting client[%d]", i)
	conn, err := net.Dial("tcp", *host)
	if err != nil {
		log.Fatal("dial: ", err)
	}
	cc := mqtt.NewClientConn(conn, 100)
	cc.ClientId = fmt.Sprintf("many.%d", i)
	cc.Dump = *dump
	cc.KeepAlive = uint16(*keepalive)

	if err := cc.Connect(*user, *pass); err != nil {
		log.Fatal("connect: %v\n", err)
		os.Exit(1)
	}

	half := int32(*pace / 2)

	log.Printf("client[%d] connected", i)

	var msgId uint16 = 1
	var maxMsgId = uint16((1 << 16) - 1)
	for {
		cc.Publish(&proto.Publish{
			Header:    proto.Header{QosLevel: proto.QosLevel(*qos)},
			TopicName: topic,
			MessageId: msgId,
			Payload:   payload,
		})
		if msgId == maxMsgId {
			msgId = 1
		}
		sltime := rand.Int31n(half) - (half / 2) + int32(*pace)
		log.Printf("client[%d] published %d, sleep %ds...", i, msgId, sltime)
		time.Sleep(time.Duration(sltime) * time.Second)
		msgId++
	}
}
