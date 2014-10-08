package config

import (
	conf "github.com/funkygao/jsconf"
)

type Config struct {
}

func LoadConfig(cf *conf.Conf) *Config {
	this := new(Config)
	return this

}
