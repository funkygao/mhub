package node

import (
	"github.com/funkygao/gomqtt/config"
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
	<-this.stopChan

}
