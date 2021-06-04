package clients

import (
	"github.com/eclipse/paho.mqtt.golang/packets"
)

type Retain struct {
	Packet *packets.PublishPacket
}

func NewRetain(p *packets.PublishPacket) *Retain {
	r := new(Retain)
	r.Packet = p
	return r
}
