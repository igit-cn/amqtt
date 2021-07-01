package broker

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/werbenhu/amqtt/config"
	"github.com/werbenhu/amqtt/ifs"
	"github.com/werbenhu/amqtt/logger"
	"github.com/werbenhu/amqtt/packets"
	"golang.org/x/net/websocket"
)

const (
	StateGapSec = 10
)

type Broker struct {
	ctx       context.Context
	cancel    context.CancelFunc
	s         ifs.Server
	processor *Processor
	ticker    *time.Ticker
}

func NewBroker(server ifs.Server) *Broker {
	b := new(Broker)
	b.s = server
	b.ctx, b.cancel = context.WithCancel(server.Context())
	b.processor = NewProcessor(server)
	return b
}

func (b *Broker) Handler(conn net.Conn, typ int) {
	client := NewClient(conn, typ)
	packet, err := client.ReadPacket()
	if err != nil {
		logger.Error("read connect packet error: ", err)
		client.Close()
		return
	}
	cp, ok := packet.(*packets.ConnectPacket)
	if !ok {
		logger.Error("received msg that was not connect")
		client.Close()
		return
	}

	b.processor.ProcessConnect(client, cp)
	client.ReadLoop(b.processor)
}

func (b *Broker) StartWebsocket() {
	http.Handle("/"+config.WsPath(), websocket.Handler(func(conn *websocket.Conn) {
		conn.PayloadType = websocket.BinaryFrame
		b.Handler(conn, config.TypWs)
	}))
	var err error
	if config.IsWsTsl() {
		err = http.ListenAndServeTLS(config.WsHost(), config.CaFile(), config.CeKey(), nil)
	} else {
		err = http.ListenAndServe(config.WsHost(), nil)
	}
	logger.Debug("StartWebsocket end")
	if err != nil {
		panic("StartWebsocket ERROR: " + err.Error())
	}
}

func (b *Broker) publishState() {
	topics := map[string]string{
		"$SYS/broker/bytes/received":            strconv.Itoa(int(b.s.State().BytesRecv)),
		"$SYS/broker/bytes/sent":                strconv.Itoa(int(b.s.State().BytesSent)),
		"$SYS/broker/clients/connected":         strconv.Itoa(int(b.s.State().ClientsConnected)),
		"$SYS/broker/clients/active":            strconv.Itoa(int(b.s.State().ClientsConnected)),
		"$SYS/broker/clients/disconnected":      strconv.Itoa(int(b.s.State().ClientsDisconnected)),
		"$SYS/broker/clients/inactive":          strconv.Itoa(int(b.s.State().ClientsDisconnected)),
		"$SYS/broker/clients/maximum":           strconv.Itoa(int(b.s.State().ClientsMax)),
		"$SYS/broker/clients/total":             strconv.Itoa(int(b.s.State().ClientsTotal)),
		"$SYS/broker/messages/inflight":         strconv.Itoa(int(b.s.State().Inflight)),
		"$SYS/broker/messages/received":         strconv.Itoa(int(b.s.State().MsgRecv)),
		"$SYS/broker/messages/sent":             strconv.Itoa(int(b.s.State().MsgSent)),
		"$SYS/broker/publish/messages/received": strconv.Itoa(int(b.s.State().PubRecv)),
		"$SYS/broker/publish/messages/sent":     strconv.Itoa(int(b.s.State().PubSent)),
		"$SYS/broker/retained_messages/count":   strconv.Itoa(int(b.s.State().Retain)),
		"$SYS/broker/store/messages/count":      strconv.Itoa(int(b.s.State().StoreCount)),
		"$SYS/broker/store/messages/bytes":      strconv.Itoa(int(b.s.State().StoreBytes)),
		"$SYS/broker/subscriptions/count":       strconv.Itoa(int(b.s.State().SubCount)),
		"$SYS/broker/version":                   b.s.State().Version,
		"$SYS/broker/uptime":                    strconv.FormatInt(time.Now().Unix()-b.s.State().Uptime, 10),
		"$SYS/broker/timestamp":                 strconv.FormatInt(time.Now().Unix(), 10),
	}
	for topic, msg := range topics {
		brokerSubs := b.s.BrokerTopics().Subscribers(topic)
		for _, sub := range brokerSubs {
			if sub != nil {
				packet := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
				packet.Retain = false
				packet.Payload = []byte(msg)
				packet.TopicName = topic
				b.processor.WritePacket(sub.(ifs.Client), packet)
			}
		}
	}
}

func (b *Broker) StartStateLoop() {
	for {
		select {
		case <-b.ticker.C:
			b.publishState()
		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Broker) StartTcp() {
	tcpHost := config.TcpHost()
	var tcpListener net.Listener
	var err error

	b.ticker = time.NewTicker(StateGapSec * time.Second)
	go b.StartStateLoop()

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
			go b.Handler(conn, config.TypTcp)
		}
	}
}

func (b *Broker) Close() {
	b.cancel()
}
