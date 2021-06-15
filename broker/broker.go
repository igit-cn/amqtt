package broker

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/ifs"
	"github.com/werbenhu/amq/logger"
)

type Broker struct {
	ctx       context.Context
	cancel    context.CancelFunc
	server    ifs.Server
	processor ifs.Processor
}

func NewBroker(server ifs.Server) *Broker {
	s := new(Broker)
	s.server = server
	s.ctx, s.cancel = context.WithCancel(server.Context())
	s.processor = NewProcessor(server)
	return s
}

func (s *Broker) Handler(conn net.Conn) {
	client := NewClient(conn, s.server.BrokerTopics())
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
	s.processor.ProcessConnect(client, cp)
	client.ReadLoop(s.processor)
}

func (s *Broker) Start() {
	tcpHost := config.TcpHost()
	var tcpListener net.Listener
	var err error

	if !config.IsTcpTsl() {
		tcpListener, err = net.Listen("tcp", tcpHost)
		if err != nil {
			logger.Fatalf("tcp listen to %s Err:%s\n", tcpHost, err)
		}
	} else {
		cert, err := tls.LoadX509KeyPair(config.CaFile(), config.CeKey())
		if err != nil {
			logger.Fatalf("tcp LoadX509KeyPair ce file: %s Err:%s\n", config.CaFile(), err)
		}
		tcpListener, err = tls.Listen("tcp", tcpHost, &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		if err != nil {
			logger.Fatalf("tsl listen to %s Err:%s\n", tcpHost, err)
		}
	}

	for {
		conn, err := tcpListener.Accept()
		if err != nil {
			logger.Fatalf("broker tcp Accept to %s Err:%s\n", tcpHost, err)
			continue
		} else {
			go s.Handler(conn)
		}
	}
}

func (s *Broker) Close() {
	s.cancel()
}
