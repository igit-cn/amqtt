package cluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/werbenhu/amqtt/config"
	"github.com/werbenhu/amqtt/ifs"
	"github.com/werbenhu/amqtt/logger"
	"github.com/werbenhu/amqtt/packets"
)

type Cluster struct {
	ctx       context.Context
	cancel    context.CancelFunc
	s         ifs.Server
	processor ifs.Processor
}

func NewCluster(server ifs.Server) *Cluster {
	c := new(Cluster)
	c.s = server
	c.ctx, c.cancel = context.WithCancel(server.Context())
	c.processor = NewProcessor(server)
	return c
}

func (c *Cluster) HandlerServer(conn net.Conn) {
	client := NewClient(conn, config.TypServer)
	packet, err := client.ReadPacket()
	if err != nil {
		logger.Error("read connect packet error: ", err)
		return
	}
	cp, ok := packet.(*packets.ConnectPacket)
	if !ok {
		logger.Error("received msg that was not connect")
		return
	}
	client.SetId(cp.ClientIdentifier)
	c.processor.ProcessConnect(client, cp)
	client.ReadLoop(c.processor)
}

func (c *Cluster) StartServer() {
	tcpHost := config.ClusterHost()
	var tcpListener net.Listener
	var err error

	if !config.IsClusterTsl() {
		tcpListener, err = net.Listen("tcp", tcpHost)
		if err != nil {
			logger.Fatalf("tcp listen to %s Err:%s", tcpHost, err)
		}
		fmt.Printf("start cluster tcp listen to %s ...\n", tcpHost)
	} else {
		cert, err := tls.LoadX509KeyPair(config.CaFile(), config.CeKey())
		if err != nil {
			logger.Fatalf("tcp LoadX509KeyPair ce file: %s Err:%s", config.CaFile(), err)
		}
		tcpListener, err = tls.Listen("tcp", tcpHost, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		if err != nil {
			logger.Fatalf("tsl listen to %s Err:%s", tcpHost, err)
		}
		fmt.Printf("start cluster tcp listen to %s and tls is on ...\n", tcpHost)
	}

	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			logger.Fatalf("cluster server tcp Accept to %s Err:%s", tcpHost, err.Error())
			continue
		} else {
			go c.HandlerServer(conn)
		}
	}
}

func (c *Cluster) HandlerClient(conn net.Conn, cluster *config.ClusterNode) {
	connect := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
	connect.ProtocolName = "MQTT"
	connect.ProtocolVersion = 4
	connect.CleanSession = true
	connect.ClientIdentifier = config.ClusterName()
	connect.Keepalive = 60

	client := NewClient(conn, config.TypClient)
	client.SetId(cluster.Name)
	err := client.WritePacket(connect)
	if err != nil {
		logger.Errorf("send cluster connect error:%s", err)
		return
	}

	if old, ok := c.s.Clusters().Load(cluster.Name); ok {
		logger.Debugf("cluster HandlerClient close clientId:%s", client.GetId())
		oldClient := old.(ifs.Client)
		oldClient.Close()
	}
	client.SetId(cluster.Name)
	c.s.Clusters().Store(cluster.Name, client)

	client.ReadLoop(c.processor)
}

func (s *Cluster) StartClient(cluster *config.ClusterNode) {
	logger.Infof("Cluster start connect to %s", cluster.Name)
	conn, err := net.DialTimeout("tcp", cluster.Host, 60*time.Second)
	if err != nil {
		logger.Errorf("Cluster fail to connect to %s Err:%s", cluster.Host, err.Error())
		return
	} else {
		s.HandlerClient(conn, cluster)
	}
}

func (c *Cluster) CheckHealthy() {
	for _, cluster := range config.Clusters() {
		clientId := strings.TrimSpace(cluster.Name)
		exist, ok := c.s.Clusters().Load(clientId)
		logger.Debugf("CheckHealthy clientId:%s, ok:%t", clientId, ok)
		if !ok {
			logger.Debugf("reconnect clientId:%s", clientId)
			go c.StartClient(&cluster)
		} else if exist.(*Client).GetTyp() == config.TypClient {
			logger.Debugf("CheckHealthy write PingreqPacket conn:%+v", exist)
			ping := packets.NewControlPacket(packets.Pingreq).(*packets.PingreqPacket)
			exist.(*Client).WritePacket(ping)
		}
	}
}

func (c *Cluster) HeartBeat() {
	tick := time.NewTicker(20 * time.Second)
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-tick.C:
			c.CheckHealthy()
		}
	}
}

func (c *Cluster) Start() {
	go c.StartServer()
	go c.CheckHealthy()
	go c.HeartBeat()

	<-c.ctx.Done()
	logger.Debug("cluster done")
}

func (c *Cluster) Close() {
	c.cancel()
}
