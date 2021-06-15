package cluster

import (
	"context"
	"errors"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/ifs"
	"github.com/werbenhu/amq/logger"
)

const (
	TYP_CLIENT = iota
	TYP_SERVER
)

type Client struct {
	id         string
	conn       net.Conn
	ctx        context.Context
	typ        int
	cancelFunc context.CancelFunc
	subscribes []string
	topics     ifs.Topic
}

func NewClient(conn net.Conn, topics ifs.Topic, typ int) ifs.Client {
	c := new(Client)
	c.conn = conn
	c.topics = topics
	c.typ = typ
	c.ctx, c.cancelFunc = context.WithCancel(context.Background())
	return c
}

func (c *Client) ReadLoop(processor ifs.Processor) {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			packet, err := c.ReadPacket()
			if err != nil {
				logger.Debugf("Client ReadLoop read packet error: %s", err.Error())
				packet := packets.NewControlPacket(packets.Disconnect).(*packets.DisconnectPacket)
				processor.ProcessMessage(c, packet)
				return
			}
			processor.ProcessMessage(c, packet)
		}
	}
}

func (c *Client) SetId(id string) ifs.Client {
	c.id = id
	return c
}

func (c *Client) SetConn(conn net.Conn) ifs.Client {
	c.conn = conn
	return c
}

func (c *Client) SetTopics(topics ifs.Topic) ifs.Client {
	c.topics = topics
	return c
}

func (c *Client) GetId() string {
	return c.id
}

func (c *Client) GetConn() net.Conn {
	return c.conn
}

func (c *Client) GetTyp() int {
	return c.typ
}

func (c *Client) GetTopics() ifs.Topic {
	return c.topics
}

func (c *Client) ReadPacket() (packets.ControlPacket, error) {
	if c.conn == nil {
		return nil, errors.New("ERR: CONN IS DISCONNECTED")
	}
	return packets.ReadPacket(c.conn)
}

func (c *Client) WritePacket(packet packets.ControlPacket) error {
	if c.conn == nil {
		return errors.New("ERR: CONN IS DISCONNECTED")
	}
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
