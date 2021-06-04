package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
)

type Cluster struct {
	Name string
	Host string
	Port int
	Tsl  bool
}

type Config struct {
	TcpHost string
	TcpPort int
	TcpTsl  bool

	WsHost string
	WsPort int
	WsTsl  bool

	CaFile string
	CeKey  string

	ClusterName string
	ClusterHost string
	ClusterPort int
	ClusterTsl  bool

	Clusters []Cluster
}

var cfg Config

func init() {
	if _, err := toml.DecodeFile("conf.toml", &cfg); err != nil {
		log.Fatalf("read conf.toml Err:%s\n", err)
		return
	}
}

func Clusters() []Cluster {
	return cfg.Clusters
}

func ClusterHost() string {
	if cfg.ClusterHost != "" {
		return fmt.Sprintf("%s:%d", cfg.ClusterHost, cfg.ClusterPort)
	}
	return ""
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
