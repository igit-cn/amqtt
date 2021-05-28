package clients

import (
	"context"
	"net"
	"sync"

	"github.com/eclipse/paho.mqtt.golang/packets"
)

type Client struct {
	ClientId   string
	Conn       net.Conn
	Subs       []string
	Ctx        context.Context
	CancelFunc context.CancelFunc
	mu         sync.Mutex
}

func NewClient(conn net.Conn) *Client {
	c := new(Client)
	c.Conn = conn
	c.Ctx, c.CancelFunc = context.WithCancel(context.Background())
	return c
}

func (c *Client) SetClientId(clientId string) *Client {
	c.ClientId = clientId
	return c
}

func (c *Client) ReadPacket() (packets.ControlPacket, error) {
	return packets.ReadPacket(c.Conn)
}

func (c *Client) WritePacket(packet packets.ControlPacket) error {
	c.mu.Lock()
	defer func(c *Client) {
		c.mu.Unlock()
	}(c)
	err := packet.Write(c.Conn)
	return err
}

func (c *Client) Close() error {
	var err error
	if c.Conn != nil {
		err = c.Conn.Close()
		c.Conn = nil
	}
	return err
}
