package ifs

import (
	"github.com/werbenhu/amqtt/packets"
)

type Processor interface {
	ProcessConnack(client Client, cp *packets.ConnackPacket)
	ProcessConnect(client Client, cp *packets.ConnectPacket)
	ProcessMessage(client Client, cp packets.ControlPacket)
}
