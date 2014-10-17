package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/funkygao/mhub/broker"
	proto "github.com/funkygao/mqttmsg"
)

var host = flag.String("host", "localhost:1883", "hostname of broker")
var id = flag.String("id", "", "client id")
var user = flag.String("user", "", "username")
var pass = flag.String("pass", "", "password")
var dump = flag.Bool("dump", false, "dump messages?")
var conns = flag.Int("conns", 500, "how many conns")

var recv int64
var abort int64
var goAhead = make(chan bool)
var dialWg sync.WaitGroup

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sub topic [topic topic...]")
		return
	}
	for i := 0; i < flag.NArg(); i++ {
		for j := 0; j < *conns; j++ {
			dialWg.Add(1)
			go subscribe(flag.Arg(i), j)
		}
	}

	dialWg.Wait()
	log.Printf("%d connections made", *conns)

	// broadcast to all goroutines, let them begin mqtt
	close(goAhead)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for _ = range ticker.C {
			log.Printf("recv: %d, abort: %d", atomic.LoadInt64(&recv),
				atomic.LoadInt64(&abort))
		}
	}()

	<-make(chan bool)
}

func subscribe(topic string, no int) {
	defer func() {
		atomic.AddInt64(&abort, 1)
	}()

	time.Sleep(time.Millisecond * time.Duration(rand.Intn(500)))
	conn, err := net.Dial("tcp", *host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		dialWg.Done()
		return
	}

	dialWg.Done()
	<-goAhead

	cc := mqtt.NewClientConn(conn, 100)
	cc.Dump = *dump
	cc.KeepAlive = 60
	cc.ClientId = fmt.Sprintf("sub.%d", no)

	if err := cc.Connect(*user, *pass); err != nil {
		log.Printf("connect: %v\n", err)
		return
	}
	//fmt.Println("Connected with client id ", cc.ClientId)

	tq := make([]proto.TopicQos, 1)
	tq[0].Topic = topic
	tq[0].Qos = proto.QosAtMostOnce
	cc.Subscribe(tq)

	for m := range cc.Incoming {
		atomic.AddInt64(&recv, 1)

		if *dump {
			fmt.Print(m.TopicName, "\t")
			m.Payload.WritePayload(os.Stdout)
			fmt.Println("\tr: ", m.Header.Retain)
		}

	}
}
