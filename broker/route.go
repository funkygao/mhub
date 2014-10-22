package broker

import (
	"strings"
)

func isGlobalTopic(topic string) bool {
	return strings.HasPrefix(topic, "System")
}
