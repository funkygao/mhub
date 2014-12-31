package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/funkygao/golib/color"
	mqtt "github.com/funkygao/mhub/broker"
	proto "github.com/funkygao/mqttmsg"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	server string
	topic  string
	user   string

	netRtt time.Duration

	onlineUsers = make(map[string]bool)

	cc *mqtt.ClientConn
)

func init() {
	//flag.StringVar(&server, "server", "192.168.22.73:1883", "broker addr")
	//flag.StringVar(&server, "server", "dw-dev.socialgamenet.com:1883", "broker addr")
	flag.StringVar(&server, "server", "localhost:1883", "broker server addr")
	flag.StringVar(&topic, "topic", "world", "subscription topic name")
	flag.StringVar(&user, "user", "", "chat username")
	flag.Parse()

	if user == "" {
		flag.Usage()
		os.Exit(0)
	}
}

func main() {
	setupChat()
	cliLoop()
}

func setupChat() {
	fmt.Println("Connecting to broker...")
	t1 := time.Now()
	conn, err := net.DialTimeout("tcp", server, time.Second*10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	netRtt = time.Since(t1)

	cc = mqtt.NewClientConn(conn, 100)
	cc.KeepAlive = 600
	if err := cc.Connect("", ""); err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}

	cc.Subscribe([]proto.TopicQos{proto.TopicQos{Topic: topic, Qos: proto.QosAtMostOnce}})
	fmt.Printf("Broker connected, subscribed to topic: %s\n", topic)
	fmt.Println("Translation engine connected: en -> zh-CN")

	go subLoop()
}

func subLoop() {
	var text string
	for m := range cc.Incoming {
		text = string(m.Payload.(proto.BytesPayload))
		fmt.Printf("[%s] -> %s\n", color.Yellow(m.TopicName), color.Red(text))
		body := strings.SplitN(text, ":", 2)
		onlineUsers[body[0]] = true
		fmt.Printf("[%s] => %s\n", color.Yellow(m.TopicName),
			color.Green(translate(body[1])))
	}
}

func translate(q string) string {
	t1 := time.Now()
	res, err := http.Get(fmt.Sprintf("http://translate.funplusgame.com/api/translate?q=%s&source=en&target=zh-CN&profanity=off", url.QueryEscape(q)))
	if err != nil {
		return err.Error()
	}
	defer res.Body.Close()

	payload, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err.Error()
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Sprintf("unexpected translation status: %s", res.Status)
	}

	return fmt.Sprintf("'%s' in %s: %s", q, time.Since(t1)-netRtt, string(payload))
}

func cliLoop() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("[%s] Enter English text: ", topic)
		text, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		text = text[:len(text)-1] // strip EOL
		if text == "" {
			continue
		}

		if handleCliCmd(text) {
			// its internal command
			continue
		}

		cc.Publish(&proto.Publish{
			Header:    proto.Header{},
			TopicName: topic,
			Payload:   proto.BytesPayload([]byte(user + ":" + text)),
		})
	}

}

func handleCliCmd(txt string) bool {
	switch txt {
	case "help":
		fmt.Println("help users rtt")
		return true

	case "users":
		for u, _ := range onlineUsers {
			fmt.Printf("%s ", u)
		}
		fmt.Println()
		return true

	case "rtt":
		fmt.Println(netRtt)
		return true
	}

	return false
}
