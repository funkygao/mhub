package discovery

import (
	"errors"
	"fmt"
	etcdErr "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
	log "github.com/funkygao/log4go"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	keyPrefix    = "/v2/keys/"
	stateKey     = "_state"
	startedState = "started"
	defaultTTL   = 604800 // One week
)

type Discoverer struct {
	client *etcd.Client

	name         string
	peer         string
	prefix       string
	discoveryURL string
}

var defaultDiscoverer *Discoverer

func init() {
	defaultDiscoverer = &Discoverer{}
}

func (this *Discoverer) Do(discoveryURL string, name string, peer string) (peers []string,
	err error) {
	this.name = name
	this.peer = peer
	this.discoveryURL = discoveryURL

	u, err := url.Parse(discoveryURL)
	if err != nil {
		return
	}

	// prefix is prepended to all keys for this discovery
	this.prefix = strings.TrimPrefix(u.Path, keyPrefix)

	// Connect to a scheme://host not a full URL with path
	log.Info("Discovery via %s using prefix %s.", u.String(), this.prefix)

	this.client = etcd.NewClient([]string{u.String()})

	// Register this machine first and announce that we are a member of
	// this cluster
	err = this.heartbeat()
	if err != nil {
		return
	}

	go this.startHeartbeat()

	// Attempt to take the leadership role, if there is no error we are it!
	resp, err := this.client.Create(path.Join(this.prefix, stateKey), startedState, 0)
	if err != nil {
		// Bail out on unexpected errors
		if clientErr, ok := err.(*etcd.EtcdError); !ok || clientErr.ErrorCode != etcdErr.EcodeNodeExist {
			return nil, err
		}
	}

	// If we got a response then the CAS was successful, we are leader
	if resp != nil && resp.Node.Value == startedState {
		// We are the leader, we have no peers
		log.Info("Discovery _state was empty, so this machine is the initial leader.")
		return nil, nil
	}

	// Fall through to finding the other discovery peers
	return this.findPeers()
}

func (this *Discoverer) findPeers() (peers []string, err error) {
	resp, err := this.client.Get(path.Join(this.prefix), false, true)
	if err != nil {
		return nil, err
	}

	node := resp.Node
	if node == nil {
		return nil, fmt.Errorf("%s key doesn't exist.", this.prefix)
	}

	for _, n := range node.Nodes {
		// Skip our own entry in the list, there is no point
		if strings.HasSuffix(n.Key, "/"+this.name) {
			continue
		}
		peers = append(peers, n.Value)
	}

	if len(peers) == 0 {
		return nil, errors.New("Discovery found an initialized cluster but no peers are registered.")
	}

	log.Info("Discovery found peers %v", peers)

	return
}

func (this *Discoverer) startHeartbeat() {
	// In case of errors we should attempt to heartbeat fairly frequently
	heartbeatInterval := defaultTTL / 8
	ticker := time.Tick(time.Second * time.Duration(heartbeatInterval))
	for {
		select {
		case <-ticker:
			err := this.heartbeat()
			if err != nil {
				log.Warn("Discovery heartbeat failed: %v", err)
			}
		}
	}
}

func (d *Discoverer) heartbeat() error {
	_, err := d.client.Set(path.Join(d.prefix, d.name), d.peer, defaultTTL)
	return err
}

func Do(discoveryURL string, name string, peer string) ([]string, error) {
	return defaultDiscoverer.Do(discoveryURL, name, peer)
}
