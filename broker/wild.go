package broker

import (
	"strings"
)

type wild struct {
	wild []string
	c    *incomingConn
}

func isWildcard(topic string) bool {
	if strings.Contains(topic, "#") || strings.Contains(topic, "+") {
		return true
	}
	return false
}

func (w wild) valid() bool {
	for i, part := range w.wild {
		// catch things like finance#
		if isWildcard(part) && len(part) != 1 {
			return false
		}
		// # can only occur as the last part
		if part == "#" && i != len(w.wild)-1 {
			return false
		}
	}
	return true
}
func newWild(topic string, c *incomingConn) wild {
	return wild{wild: strings.Split(topic, "/"), c: c}
}

func (w wild) matches(parts []string) bool {
	i := 0
	for i < len(parts) {
		// topic is longer, no match
		if i >= len(w.wild) {
			return false
		}
		// matched up to here, and now the wildcard says "all others will match"
		if w.wild[i] == "#" {
			return true
		}
		// text does not match, and there wasn't a + to excuse it
		if parts[i] != w.wild[i] && w.wild[i] != "+" {
			return false
		}
		i++
	}

	// make finance/stock/ibm/# match finance/stock/ibm
	if i == len(w.wild)-1 && w.wild[len(w.wild)-1] == "#" {
		return true
	}

	if i == len(w.wild) {
		return true
	}
	return false
}
