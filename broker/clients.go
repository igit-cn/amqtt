package broker

import (
	"context"
	"errors"
	"net"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amqtt/ifs"
	"github.com/werbenhu/amqtt/logger"
)

type Client struct {
	id         string
	conn       net.Conn
	ctx        context.Context
	cancelFunc context.CancelFunc
	subscribes []string
	topics     ifs.Topic
	typ        int
	will       *packets.PublishPacket
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
				logger.Debugf("Client ReadLoop read packet error: %+v\n", err)
				processor.ProcessMessage(c, c.will)
				packet := packets.NewControlPacket(packets.Disconnect).(*packets.DisconnectPacket)
				processor.ProcessMessage(c, packet)
				return
			}
			logger.Debug("ReadLoop success")
			processor.ProcessMessage(c, packet)
		}
	}
}

func (c *Client) SetWill(will *packets.PublishPacket) {
	c.will = will
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

func (c *Client) GetTopics() ifs.Topic {
	return c.topics
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
	if c.conn != nil {
		logger.Debugf("WritePacket id:%s, packet:%s", c.id, packet.String())
		err := packet.Write(c.conn)
		return err
	}
	return nil
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
