package node

import (
	"github.com/funkygao/gomqtt/config"
)

type Node struct {
	cf *config.Config
}

func New(cf *config.Config) (this *Node) {
	this = new(Node)
	this.cf = cf
	return
}

func (this *Node) ServeForever() {

}
