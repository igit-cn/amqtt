package config

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/BurntSushi/toml"
)

type ClusterNode struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

type Config struct {
	TcpHost string
	TcpPort int
	TcpTsl  bool

	WsHost string
	WsPort int
	WsPath string
	WsTsl  bool

	CaFile string
	CeKey  string

	ClusterName string
	ClusterHost string
	ClusterPort int
	ClusterTsl  bool

	Clusters []ClusterNode
}

var cfg Config

func init() {
	if _, err := toml.DecodeFile("conf.toml", &cfg); err != nil {
		log.Fatalf("read conf.toml Err:%s\n", err)
		return
	}
}

func Configure(args []string) error {
	fs := flag.NewFlagSet("amq", flag.ExitOnError)

	var clusters string

	fs.IntVar(&cfg.TcpPort, "p", 1884, "Tcp port to listen on.")
	fs.StringVar(&cfg.TcpHost, "host", "0.0.0.0", "Tcp host to listen on.")
	fs.IntVar(&cfg.ClusterPort, "cp", 1882, "Cluster tcp port to listen on.")
	fs.StringVar(&cfg.ClusterHost, "ch", "0.0.0.0", "Cluster tcp host to listen on.")
	fs.StringVar(&clusters, "clusters", "", "Cluster list.")

	fs.StringVar(&cfg.ClusterName, "name", "node01", "Cluster node name.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if clusters != "" {
		items := strings.Split(clusters, ",")
		cfg.Clusters = []ClusterNode{}
		for _, cluster := range items {
			parts := strings.Split(cluster, "//")
			if len(parts) != 2 {
				log.Fatalf("read conf.toml clusters param error\n")
			}
			cfg.Clusters = append(cfg.Clusters, ClusterNode{
				Name: parts[0],
				Host: parts[1],
			})
		}
	}
	return nil
}

func Clusters() []ClusterNode {
	return cfg.Clusters
}

func ClusterName() string {
	return cfg.ClusterName
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

func WsPath() string {
	return cfg.WsPath
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
