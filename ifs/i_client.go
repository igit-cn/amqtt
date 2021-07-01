package ifs

import (
	"net"

	"github.com/werbenhu/amqtt/packets"
)

type Client interface {
	ReadLoop(processor Processor)
	SetId(id string) Client
	GetId() string
	SetConn(conn net.Conn) Client
	GetConn() net.Conn
	ReadPacket() (packets.ControlPacket, error)
	WritePacket(packet packets.ControlPacket) error
	Topics() map[string]interface{}
	AddTopic(topic string, data interface{}) (exist bool)
	RemoveTopic(topic string) error
	Done() <-chan struct{}
	Close() error
	GetTyp() int
}
