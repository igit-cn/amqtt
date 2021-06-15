package cluster

import (
	"context"
	"crypto/tls"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/ifs"
	"github.com/werbenhu/amq/logger"
	"net"
	"strings"
	"time"
)

type Cluster struct {
	ctx       context.Context
	cancel    context.CancelFunc
	server    ifs.Server
	processor ifs.Processor
}

func NewCluster(server ifs.Server) *Cluster {
	s := new(Cluster)
	s.server = server
	s.ctx, s.cancel = context.WithCancel(server.Context())
	s.processor = NewProcessor(server)
	return s
}

func (s *Cluster) HandlerServer(conn net.Conn) {
	client := NewClient(conn, s.server.ClusterTopics(), TYP_SERVER)
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
	s.processor.ProcessConnect(client, cp)
	client.ReadLoop(s.processor)
}

func (s *Cluster) StartServer() {
	tcpHost := config.ClusterHost()
	var tcpListener net.Listener
	var err error

	if !config.IsTcpTsl() {
		tcpListener, err = net.Listen("tcp", tcpHost)
		if err != nil {
			logger.Fatalf("tcp listen to %s Err:%s", tcpHost, err)
		}
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
	}

	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			logger.Fatalf("cluster server tcp Accept to %s Err:%s", tcpHost, err.Error())
			continue
		} else {
			go s.HandlerServer(conn)
		}
	}
}

func (s *Cluster) HandlerClient(conn net.Conn, cluster *config.ClusterNode) {
	connect := packets.NewControlPacket(packets.Connect).(*packets.ConnectPacket)
	connect.ProtocolName = "MQTT"
	connect.ProtocolVersion = 4
	connect.CleanSession = true
	connect.ClientIdentifier = config.ClusterName()
	connect.Keepalive = 60

	client := NewClient(conn, s.server.ClusterTopics(), TYP_CLIENT)
	client.SetId(cluster.Name)
	err := client.WritePacket(connect)
	if err != nil {
		logger.Errorf("send cluster connect error:%s", err)
		return
	}

	if old, ok := s.server.ClusterClients().Load(cluster.Name); ok {
		oldClient := old.(ifs.Client)
		oldClient.Close()
	}
	logger.Debugf("cluster ProcessConnack clientId:%s", client.GetId())
	client.SetId(cluster.Name)
	s.server.ClusterClients().Store(cluster.Name, client)

	client.ReadLoop(s.processor)
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

func (s *Cluster) CheckHealthy() {
	for _, cluster := range config.Clusters() {
		clientId := strings.TrimSpace(cluster.Name)
		exist, ok := s.server.ClusterClients().Load(clientId)
		logger.Debugf("CheckHealthy clientId:%s, ok:%t", clientId, ok)
		if !ok {
			logger.Debugf("reconnect clientId:%s", clientId)
			go s.StartClient(&cluster)
		} else if exist.(*Client).GetTyp() == TYP_CLIENT {
			logger.Debugf("CheckHealthy write PingreqPacket conn:%+v", exist)
			ping := packets.NewControlPacket(packets.Pingreq).(*packets.PingreqPacket)
			exist.(*Client).WritePacket(ping)
		}
	}
}

func (s *Cluster) HeartBeat() {
	tick := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tick.C:
			s.CheckHealthy()
		}
	}
}

func (s *Cluster) Start() {
	go s.StartServer()
	go s.CheckHealthy()
	go s.HeartBeat()

	<-s.ctx.Done()
	logger.Debug("cluster done")
}

func (s *Cluster) Close() {
	s.cancel()
}
