/*
Simulates a number of pairs of clients where one is sending as fast as possible to the other.
Realistically, this ends up testing the ability of the system to decode and queue messages, because
any slight inbalance in scheduling of readers causes a pile up of messages from the writer slamming
them down the throat of the server.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	mqtt "github.com/funkygao/mhub/broker"
	proto "github.com/funkygao/mqttmsg"
)

var conns = flag.Int("conns", 100, "how many connections")
var messages = flag.Int("messages", 100, "how many messages")
var host = flag.String("host", "localhost:1883", "hostname of broker")
var dump = flag.Bool("dump", false, "dump messages?")
var id = flag.String("id", "", "client id")
var user = flag.String("user", "", "username")
var pace = flag.Int("pace", 10, "send a message on average once every pace seconds")
var pass = flag.String("pass", "", "password")

var cwg sync.WaitGroup

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	flag.Parse()

	timeStart := time.Now()

	// a system to check how long connection establishment takes
	cwg.Add(*conns)
	go func() {
		cwg.Wait()
		log.Print("all connections made")
	}()

	var wg sync.WaitGroup
	publishers := *conns / 2
	for i := 0; i < publishers; i++ {
		wg.Add(2)
		go func(i int) {
			sub(i, &wg)
			pub(i)

			wg.Done()
		}(i)
	}
	log.Print("all started")
	wg.Wait()
	log.Print("all finished")

	timeEnd := time.Now()

	rc := 0
loop:
	for {
		select {
		case i := <-bad:
			println("subscriber missed messages: ", i)
			rc = 1
		default:
			// nothing to read on bad, done.
			break loop
		}
	}

	elapsed := timeEnd.Sub(timeStart)
	totmsg := float64(*messages * publishers)
	nspermsg := float64(elapsed) / totmsg
	msgpersec := int(1 / nspermsg * 1e9)

	log.Print("elapsed time: ", elapsed)
	log.Print("messages    : ", totmsg)
	log.Print("messages/sec: ", msgpersec)

	os.Exit(rc)
}

func pub(i int) {
	topic := fmt.Sprintf("loadtest/%v", i)

	var cc *mqtt.ClientConn
	if cc = connect(); cc == nil {
		log.Println(i, " failed connection")
		return
	}

	half := int32(*pace / 2)
	for i := 0; i < *messages; i++ {
		cc.Publish(&proto.Publish{
			Header:    proto.Header{QosLevel: proto.QosAtLeastOnce},
			TopicName: topic,
			MessageId: uint16(i + 1),
			Payload:   proto.BytesPayload([]byte(`{"payload":{"330":{"uid":53,"march_id":330,"city_id":53,"opp_uid":0,"world_id":1,"type":"encamp","start_x":72,"start_y":64,"end_x":80,"end_y":78,"start_time":1412999095,"end_time":1412999111,"speed":1,"state":"marching","alliance_id":0}}`)),
		})

		if *pace > 0 {
			sltime := rand.Int31n(half) - (half / 2) + int32(*pace)
			time.Sleep(time.Duration(sltime) * time.Second)
		}
	}

	cc.Disconnect()
}

func connect() *mqtt.ClientConn {
	conn, err := net.Dial("tcp", *host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		return nil
	}
	cc := mqtt.NewClientConn(conn, 100)
	cc.Dump = *dump
	cc.KeepAlive = 60
	cc.ClientId = *id

	err = cc.Connect(*user, *pass)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	cwg.Done()
	return cc
}

// A channel to communicate subscribers that didn't get what they expected
var bad = make(chan int)

func sub(i int, wg *sync.WaitGroup) {
	topic := fmt.Sprintf("loadtest/%v", i)

	var cc *mqtt.ClientConn
	if cc = connect(); cc == nil {
		return
	}

	ack := cc.Subscribe([]proto.TopicQos{
		{topic, proto.QosAtLeastOnce},
	})
	if *dump {
		fmt.Printf("suback: %#v\n", ack)
	}

	go func() {
		ok := false
		count := 0
		for _ = range cc.Incoming {
			count++
			if count == *messages {
				cc.Disconnect()
				ok = true
			}
		}

		if !ok {
			bad <- i
		}

		wg.Done()
	}()
}
