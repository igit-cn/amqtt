package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
)

type Config struct {
	TcpHost string
	TcpPort int
	TcpTsl  bool

	WsHost string
	WsPort int
	WsTsl  bool

	CaFile string
	CeKey  string
}

var cfg Config

func init() {
	if _, err := toml.DecodeFile("conf.toml", &cfg); err != nil {
		log.Fatalf("read conf.toml Err:%s\n", err)
		return
	}
}

func TcpHost() string {
	return fmt.Sprintf("%s:%d", cfg.TcpHost, cfg.TcpPort)
}

func WsHost() string {
	return fmt.Sprintf("%s:%d", cfg.WsHost, cfg.WsPort)
}

func CaFile() string {
	return cfg.CaFile
}

func CeKey() string {
	return cfg.CeKey
}

func IsTcpTsl() bool {
	return cfg.TcpTsl
}

func IsWsTsl() bool {
	return cfg.WsTsl
}
