package node

import (
	"github.com/funkygao/gomqtt/config"
	log "github.com/funkygao/log4go"
)

type Node struct {
	cf       *config.Config
	stopChan chan bool
}

func New(cf *config.Config) (this *Node) {
	this = new(Node)
	this.cf = cf
	this.stopChan = make(chan bool)
	return
}

func (this *Node) ServeForever() {
	log.Info("starting node: %+v", *this)

	this.joinCluster()

	this.serveMqtt()

	<-this.stopChan
}
