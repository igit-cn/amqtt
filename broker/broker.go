package broker

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
)

type Broker struct {
	ctx       context.Context
	cancel    context.CancelFunc
	server    ifs.Server
	processor *Processor
}

func NewBroker(server ifs.Server) *Broker {
	s := new(Broker)
	s.server = server
	s.ctx, s.cancel = context.WithCancel(server.Context())
	s.processor = NewProcessor(server)
	return s
}

func (s *Broker) ReadLoop(client *clients.Client) {
	for {
		select {
		case <-client.Done():
			return
		default:
			packet, err := client.ReadPacket()
			if err != nil {
				fmt.Printf("read packet error: %+v\n", err)
				packet := packets.NewControlPacket(packets.Disconnect).(*packets.DisconnectPacket)
				s.processor.ProcessMessage(client, packet)
				return
			}
			s.processor.ProcessMessage(client, packet)
		}
	}
}

func (s *Broker) Handler(conn net.Conn) {
	client := clients.NewClient(conn, s.server.BrokerTopics())
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
	s.processor.ProcessConnect(client, cp)
	s.ReadLoop(client)
}

func (s *Broker) Start() {
	tcpHost := config.TcpHost()
	var tcpListener net.Listener
	var err error

	if !config.IsTcpTsl() {
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
			log.Fatalf("broker tcp Accept to %s Err:%s\n", tcpHost, err)
			continue
		} else {
			go s.Handler(conn)
		}
	}
}
