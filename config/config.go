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
	TcpTls  bool

	Websocket bool
	WsHost    string
	WsPort    int
	WsPath    string
	WsTls     bool

	Ca             string
	CertFile       string
	KeyFile        string
	ClientCertFile string
	ClientKeyFile  string

	IsCluster   bool
	ClusterName string
	ClusterHost string
	ClusterPort int
	ClusterTls  bool

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
	fs := flag.NewFlagSet("amqtt", flag.ExitOnError)

	var configFile string
	fs.StringVar(&configFile, "c", "", "the config file path.")

	fs.IntVar(&cfg.TcpPort, "port", cfg.TcpPort, "broker tcp port to listen on.")
	fs.StringVar(&cfg.TcpHost, "host", cfg.TcpHost, "broker tcp host to listen on.")

	fs.BoolVar(&cfg.TcpTls, "tls", cfg.ClusterTls, "whether broker tcp use tls")
	fs.StringVar(&cfg.Ca, "ca", cfg.Ca, "path of tls root ca file.")
	fs.StringVar(&cfg.CertFile, "certfile", cfg.CertFile, "path of tls server cert file.")
	fs.StringVar(&cfg.KeyFile, "keyfile", cfg.KeyFile, "path of tls server key file.")

	fs.IntVar(&cfg.ClusterPort, "cp", cfg.ClusterPort, "cluster tcp port to listen on.")
	fs.StringVar(&cfg.ClusterHost, "ch", cfg.ClusterHost, "cluster tcp host to listen on.")
	fs.BoolVar(&cfg.ClusterTls, "cluster-tls", cfg.ClusterTls, "whether cluster tcp use tls")

	fs.StringVar(&cfg.ClientCertFile, "client-certfile", cfg.ClientCertFile, "path of tls client cert file.")
	fs.StringVar(&cfg.ClientKeyFile, "client-keyfile", cfg.ClientKeyFile, "path of tls client key file.")

	fs.BoolVar(&cfg.IsCluster, "cluster", cfg.IsCluster, "whether to open cluster mode.")
	fs.StringVar(&cfg.ClusterName, "name", cfg.ClusterName, "current node name of the cluster.")
	clusters := *fs.String("clusters", "", "other nodes of this cluster. e.g., \"node2//host:port,node3//host:port\"")

	fs.BoolVar(&cfg.Websocket, "ws", cfg.Websocket, "whether to open websocket")
	fs.BoolVar(&cfg.Websocket, "ws-tls", cfg.WsTls, "whether websocket use tls")
	fs.StringVar(&cfg.WsPath, "ws-path", cfg.WsPath, "websocket server path. e.g., \"/mqtt\"")
	fs.IntVar(&cfg.WsPort, "ws-port", cfg.WsPort, "websocket port to listen on.")
	fs.StringVar(&cfg.WsHost, "ws-host", cfg.WsHost, "websocket host to listen on.")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if configFile != "" {
		if _, err := toml.DecodeFile(configFile, &cfg); err != nil {
			log.Fatalf("read %s Err:%s\n", configFile, err)
			return nil
		}
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

func IsWebsocket() bool {
	return cfg.Websocket
}

func WsHost() string {
	return fmt.Sprintf("%s:%d", cfg.WsHost, cfg.WsPort)
}

func IsCluster() bool {
	return cfg.IsCluster
}

func Ca() string {
	return cfg.Ca
}

func CertFile() string {
	return cfg.CertFile
}

func KeyFile() string {
	return cfg.KeyFile
}

func ClientCertFile() string {
	return cfg.ClientCertFile
}

func ClientKeyFile() string {
	return cfg.ClientKeyFile
}

func WsPath() string {
	return cfg.WsPath
}

func TcpTls() bool {
	return cfg.TcpTls
}

func WsTls() bool {
	return cfg.WsTls
}

func ClusterTls() bool {
	return cfg.ClusterTls
}
