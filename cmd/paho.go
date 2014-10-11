package main

import (
  "fmt"
  MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
  "os"
  "time"
)

//define a function for the default message handler
var f MQTT.MessageHandler = func(msg MQTT.Message) {
  fmt.Printf("TOPIC: %s\n", msg.Topic())
  fmt.Printf("MSG: %s\n", msg.Payload())
}

func main() {
  //create a ClientOptions struct setting the broker address, clientid, turn
  //off trace output and set the default message handler
  opts := MQTT.NewClientOptions().SetBroker("tcp://iot.eclipse.org:1883")
  opts.SetClientId("go-simple")
  opts.SetTraceLevel(MQTT.Off)
  opts.SetDefaultPublishHandler(f)

  //create and start a client using the above ClientOptions
  c := MQTT.NewClient(opts)
  _, err := c.Start()
  if err != nil {
    panic(err)
  }

  //subscribe to the topic /go-mqtt/sample and request messages to be delivered
  //at a maximum qos of zero, wait for the receipt to confirm the subscription
  if receipt, err := c.StartSubscription(nil, "/go-mqtt/sample", MQTT.QOS_ZERO); err != nil {
    fmt.Println(err)
    os.Exit(1)
  } else {
    <-receipt
  }

  //Publish 5 messages to /go-mqtt/sample at qos 1 and wait for the receipt
  //from the server after sending each message
  for i := 0; i < 5; i++ {
    text := fmt.Sprintf("this is msg #%d!", i)
    receipt := c.Publish(MQTT.QOS_ONE, "/go-mqtt/sample", text)
    <-receipt
  }

  time.Sleep(3 * time.Second)

  //unsubscribe from /go-mqtt/sample
  if receipt, err := c.EndSubscription("/go-mqtt/sample"); err != nil {
    fmt.Println(err)
    os.Exit(1)
  } else {
    <-receipt
  }

  c.Disconnect(250)
}
