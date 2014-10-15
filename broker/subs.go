package broker

import (
	log "github.com/funkygao/log4go"
	proto "github.com/funkygao/mqttmsg"
	"strings"
	"sync"
)

type subscriptions struct {
	posts     chan post
	mu        sync.Mutex                 // guards access to fields below
	subs      map[string][]*incomingConn // key is topic
	retain    map[string]retain          // key is topic
	wildcards []wild
	stats     *stats
}

func newSubscriptions(workers int, queueLen int, stats *stats) *subscriptions {
	this := &subscriptions{
		subs:   make(map[string][]*incomingConn),
		retain: make(map[string]retain),
		posts:  make(chan post, queueLen),
		stats:  stats,
	}
	for i := 0; i < workers; i++ {
		go this.run(i)
	}
	return this
}

func (this *subscriptions) submit(m *proto.Publish) {
	this.posts <- post{m: m}
}

func (this *subscriptions) run(id int) {
	log.Debug("subs worker[%d] started", id)

	for post := range this.posts {
		// Remember the original retain setting, but send out immediate
		// copies without retain: "When a server sends a PUBLISH to a client
		// as a result of a subscription that already existed when the
		// original PUBLISH arrived, the Retain flag should not be set,
		// regardless of the Retain flag of the original PUBLISH.
		isRetain := post.m.Header.Retain
		post.m.Header.Retain = false

		// Handle "retain with payload size zero = delete retain".
		// Once the delete is done, return instead of continuing.
		if isRetain && post.m.Payload.Size() == 0 {
			this.mu.Lock()
			delete(this.retain, post.m.TopicName)
			this.mu.Unlock()
			return
		}

		// Find all the connections that should be notified of this message
		// and queue the outgoing message
		for _, c := range this.subscribers(post.m.TopicName) {
			if c != nil {
				c.submit(post.m)
			}
		}

		if isRetain {
			this.mu.Lock()
			// Save a copy of it, and set that copy's Retain to true, so that
			// when we send it out later we notify new subscribers that this
			// is an old message.
			msg := *post.m
			msg.Header.Retain = true
			this.retain[post.m.TopicName] = retain{m: msg}
			this.mu.Unlock()
		}
	}
}

func (this *subscriptions) sendRetain(topic string, c *incomingConn) {
	this.mu.Lock()
	var tlist []string
	if isWildcard(topic) {
		// TODO: select matching topics from the retain map
	} else {
		tlist = []string{topic}
	}
	for _, t := range tlist {
		if r, ok := this.retain[t]; ok {
			c.submit(&r.m)
		}
	}
	this.mu.Unlock()
}

func (this *subscriptions) add(topic string, c *incomingConn) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if isWildcard(topic) {
		w := newWild(topic, c)
		if w.valid() {
			this.wildcards = append(this.wildcards, w)
		}
	} else {
		this.subs[topic] = append(this.subs[topic], c)
	}
}

// Find all connections that are subscribed to this topic.
func (this *subscriptions) subscribers(topic string) []*incomingConn {
	this.mu.Lock()
	defer this.mu.Unlock()

	// non-wildcard subscribers
	res := this.subs[topic]

	// process wildcards
	parts := strings.Split(topic, SLASH)
	for _, w := range this.wildcards {
		if w.matches(parts) {
			res = append(res, w.c)
		}
	}

	return res
}

// Remove all subscriptions that refer to a connection.
func (this *subscriptions) unsubAll(c *incomingConn) {
	this.mu.Lock()
	for _, v := range this.subs {
		for i := range v {
			if v[i] == c {
				v[i] = nil
			}
		}
	}

	// remove any associated entries in the wildcard list
	var wildNew []wild
	for i := 0; i < len(this.wildcards); i++ {
		if this.wildcards[i].c != c {
			wildNew = append(wildNew, this.wildcards[i])
		}
	}
	this.wildcards = wildNew

	this.mu.Unlock()
}

// Remove the subscription to topic for a given connection.
func (this *subscriptions) unsub(topic string, c *incomingConn) {
	this.mu.Lock()
	if subs, ok := this.subs[topic]; ok {
		nils := 0

		// Search the list, removing references to our connection.
		// At the same time, count the nils to see if this list is now empty.
		for i := 0; i < len(subs); i++ {
			if subs[i] == c {
				subs[i] = nil
			}
			if subs[i] == nil {
				nils++
			}
		}

		if nils == len(subs) {
			delete(this.subs, topic)
		}
	}

	this.mu.Unlock()
}
