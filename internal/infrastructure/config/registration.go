package config

import (
	"net"
	"strconv"
)

type RegistrationConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

func (c RegistrationConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}
