package config

import (
	conf "github.com/funkygao/jsconf"
)

var (
	Conf *Config
)

type Config struct {
}

func init() {
	Conf = new(Config)
}

func LoadConfig(cf *conf.Conf) {

}
