package ifs

import (
	"github.com/eclipse/paho.mqtt.golang/packets"
	"net"
)

type Client interface {
	ReadLoop(processor Processor)
	SetId(id string) Client
	GetId() string
	SetConn(conn net.Conn) Client
	GetConn() net.Conn
	SetTopics(topics Topic) Client
	GetTopics() Topic
	ReadPacket() (packets.ControlPacket, error)
	WritePacket(packet packets.ControlPacket) error
	ClearSubscribes() error
	Done() <-chan struct{}
	Close() error
	GetTyp() int
}
