package cluster

import (
	"context"
	"errors"
	"net"

	"github.com/werbenhu/amqtt/ifs"
	"github.com/werbenhu/amqtt/logger"
	"github.com/werbenhu/amqtt/packets"
)

type Client struct {
	id         string
	conn       net.Conn
	ctx        context.Context
	typ        int
	cancelFunc context.CancelFunc
	topics     map[string]interface{}
}

func NewClient(conn net.Conn, typ int) ifs.Client {
	c := new(Client)
	c.conn = conn
	c.typ = typ
	c.topics = make(map[string]interface{})
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

func (c *Client) GetId() string {
	return c.id
}

func (c *Client) GetConn() net.Conn {
	return c.conn
}

func (c *Client) GetTyp() int {
	return c.typ
}

func (c *Client) ReadPacket() (packets.ControlPacket, error) {
	if c.conn != nil {
		packet, err := packets.ReadPacket(c.conn)
		if err == nil {
			logger.Debugf("ReadPacket id:%s, packet:%s", c.id, packet.String())
		}
		return packet, err
	}
	return nil, errors.New("conn is disconnected")
}

func (c *Client) WritePacket(packet packets.ControlPacket) error {
	if c.conn == nil {
		return errors.New("ERR: CONN IS DISCONNECTED")
	}
	err := packet.Write(c.conn)
	return err
}

func (c *Client) Topics() map[string]interface{} {
	return c.topics
}

func (c *Client) AddTopic(topic string, data interface{}) (exist bool) {
	if _, ok := c.topics[topic]; ok {
		exist = true
	}
	c.topics[topic] = data
	return
}

func (c *Client) RemoveTopic(topic string) error {
	if _, ok := c.topics[topic]; !ok {
		return errors.New("topic not exist")
	}
	delete(c.topics, topic)
	return nil
}

func (c *Client) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Client) Close() error {
	c.cancelFunc()

	var err error
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}
	return err
}
