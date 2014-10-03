package mqtt

import (
	"sync"
)

type Topic struct {
	Content         string
	RetainedMessage *MqttMessage
}

func CreateTopic(content string) *Topic {
	topic := new(Topic)
	topic.Content = content
	topic.RetainedMessage = nil
	return topic
}
