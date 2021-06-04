package clients

import (
	"context"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/ifs"
	"net"
)

type Client struct {
	id         string
	conn       net.Conn
	ctx        context.Context
	cancelFunc context.CancelFunc
	subscribes []string
	topics     ifs.Topic
}

func NewClient(conn net.Conn, topics ifs.Topic) *Client {
	c := new(Client)
	c.conn = conn
	c.topics = topics
	c.ctx, c.cancelFunc = context.WithCancel(context.Background())
	return c
}

func (c *Client) SetId(id string) *Client {
	c.id = id
	return c
}

func (c *Client) SetConn(conn net.Conn) *Client {
	c.conn = conn
	return c
}

func (c *Client) SetTopics(topics ifs.Topic) *Client {
	c.topics = topics
	return c
}

func (c *Client) GetId() string {
	return c.id
}

func (c *Client) GetConn() net.Conn {
	return c.conn
}

func (c *Client) GetTopics() ifs.Topic {
	return c.topics
}

func (c *Client) ReadPacket() (packets.ControlPacket, error) {
	return packets.ReadPacket(c.conn)
}

func (c *Client) WritePacket(packet packets.ControlPacket) error {
	err := packet.Write(c.conn)
	return err
}

func (c *Client) ClearSubscribes() error {
	for _, topic := range c.subscribes {
		c.topics.Unsubscribe(topic, c.GetId())
	}
	return nil
}

func (c *Client) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Client) Close() error {
	c.cancelFunc()
	c.ClearSubscribes()

	var err error
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}
	return err
}
