package cluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/clients"
	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/ifs"
	"log"
	"net"
	"time"
)

type Cluster struct {
	ctx       context.Context
	cancel    context.CancelFunc
	server    ifs.Server
	processor *Processor
}

func NewCluster(server ifs.Server) *Cluster {
	s := new(Cluster)
	s.server = server
	s.ctx, s.cancel = context.WithCancel(server.Context())
	s.processor = NewProcessor(server)
	return s
}

func (s *Cluster) ReadLoop(client *clients.Client) {
	for {
		select {
		default:
			packet, err := client.ReadPacket()
			if err != nil {
				fmt.Printf("cluster ReadLoop read packet error: %+v\n", err)
				packet := packets.NewControlPacket(packets.Disconnect).(*packets.DisconnectPacket)
				s.processor.ProcessMessage(client, packet)
				return
			}
			s.processor.ProcessMessage(client, packet)
		}
	}
}

func (s *Cluster) HandlerServer(conn net.Conn) {
	client := clients.NewClient(conn, s.server.ClusterTopics())
	packet, err := client.ReadPacket()
	if err != nil {
		fmt.Println("read connect packet error: ", err)
		return
	}
	cp, ok := packet.(*packets.ConnectPacket)
	if !ok {
		fmt.Println("received msg that was not connect")
		return
	}
	client.SetId(cp.ClientIdentifier)
	s.processor.ProcessConnect(client, cp)
	s.ReadLoop(client)
}

func (s *Cluster) StartServer() {
	tcpHost := config.ClusterHost()
	var tcpListener net.Listener
	var err error

	if !config.IsTcpTsl() {
		fmt.Printf("start cluster server:%s\n", tcpHost)
		tcpListener, err = net.Listen("tcp", tcpHost)
		if err != nil {
			log.Fatalf("tcp listen to %s Err:%s\n", tcpHost, err)
		}
	} else {
		cert, err := tls.LoadX509KeyPair(config.CaFile(), config.CeKey())
		if err != nil {
			log.Fatalf("tcp LoadX509KeyPair ce file: %s Err:%s\n", config.CaFile(), err)
		}
		tcpListener, err = tls.Listen("tcp", tcpHost, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		if err != nil {
			log.Fatalf("tsl listen to %s Err:%s\n", tcpHost, err)
		}
	}

	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			log.Fatalf("cluster server tcp Accept to %s Err:%s\n", tcpHost, err)
			continue
		} else {
			go s.HandlerServer(conn)
		}
	}
}

func (s *Cluster) HandlerClient(conn net.Conn) {
	connect := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
	connect.ProtocolName = "MQTT"
	connect.ProtocolVersion = 4
	connect.CleanSession = true
	connect.ClientIdentifier = conn.RemoteAddr().String()
	connect.Keepalive = 60

	client := clients.NewClient(conn, s.server.ClusterTopics())
	client.SetId(connect.ClientIdentifier)
	err := client.WritePacket(connect)
	if err != nil {
		fmt.Printf("send cluster connect error:%s\n", err)
		return
	}
	s.ReadLoop(client)
}

func (s *Cluster) StartClient() {
	for _, clu := range config.Clusters() {
		hostPort := fmt.Sprintf("%s:%d", clu.Host, clu.Port)
		conn, err := net.DialTimeout("tcp", hostPort, 60 * time.Second)
		if err != nil {
			fmt.Printf("cluster client tcp Accept to %s Err:%s\n", hostPort, err)
			return
		}
		go s.HandlerClient(conn)
	}
	select {
	case <- s.ctx.Done():
		fmt.Printf("cluster client done")
		return
	}
}

func (s *Cluster) Start() {
	go s.StartServer()
	go s.StartClient()

	select {
	case <- s.ctx.Done():
		fmt.Printf("cluster done")
		return
	}
}
