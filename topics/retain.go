package topics

import (
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/clients"
)

type RetainPacket struct {
	Packet *packets.PublishPacket
	// clients map[string]*clients.Client
	mu sync.RWMutex
}

func NewRetain(p *packets.PublishPacket) *RetainPacket {
	r := new(RetainPacket)
	r.Packet = p
	// r.clients = make(map[string]*clients.Client)
	return r
}

func (r *RetainPacket) Add(client *clients.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// r.clients[client.ClientId] = client
}

func (r *RetainPacket) Del(clientId string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// delete(r.clients, clientId)
}
