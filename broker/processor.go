package broker

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/werbenhu/amqtt/config"
	"github.com/werbenhu/amqtt/ifs"
	"github.com/werbenhu/amqtt/logger"
	"github.com/werbenhu/amqtt/packets"
)

const (
	Qos2Timeout int64 = 20
)

type Processor struct {
	s           ifs.Server
	qos2Msgs    map[uint16]int64
	maxQos2Msgs int
}

func NewProcessor(server ifs.Server) *Processor {
	p := new(Processor)
	p.s = server
	p.qos2Msgs = make(map[uint16]int64)
	return p
}

func (p *Processor) checkQos2Msg(messageId uint16) error {
	if _, found := p.qos2Msgs[messageId]; found {
		delete(p.qos2Msgs, messageId)
	} else {
		return errors.New("RC_PACKET_IDENTIFIER_NOT_FOUND")
	}
	return nil
}

func (p *Processor) saveQos2Msg(messageId uint16) error {
	if p.isQos2MsgFull() {
		return errors.New("DROPPED_QOS2_PACKET_FOR_TOO_MANY_AWAITING_REL")
	}

	if _, found := p.qos2Msgs[messageId]; found {
		return errors.New("RC_PACKET_IDENTIFIER_IN_USE")
	}
	p.qos2Msgs[messageId] = time.Now().Unix()
	time.AfterFunc(time.Duration(Qos2Timeout)*time.Second, p.expireQos2Msg)
	return nil
}

func (p *Processor) expireQos2Msg() {
	if len(p.qos2Msgs) == 0 {
		return
	}
	now := time.Now().Unix()
	for messageId, ts := range p.qos2Msgs {
		if now-ts >= Qos2Timeout {
			delete(p.qos2Msgs, messageId)
		}
	}
}

func (p *Processor) isQos2MsgFull() bool {
	if p.maxQos2Msgs == 0 {
		return false
	}
	if len(p.qos2Msgs) < p.maxQos2Msgs {
		return false
	}
	return true
}

func (p *Processor) DoPublish(topic string, packet *packets.PublishPacket) {
	brokerSubs := p.s.BrokerTopics().Subscribers(topic)
	atomic.AddInt64(&p.s.State().PubRecv, 1)

	//a message is only sent to a client once, here to remove the duplicate
	brokerHistory := make(map[string]bool)
	for _, sub := range brokerSubs {
		client := sub.(ifs.Client)
		if !brokerHistory[client.GetId()] {
			brokerHistory[client.GetId()] = true
			client.WritePacket(packet)
		}
	}

	clusterSubs := p.s.ClusterTopics().Subscribers(topic)
	clusterrHistory := make(map[string]bool)
	for _, sub := range clusterSubs {
		client := sub.(ifs.Client)
		if !clusterrHistory[client.GetId()] {
			clusterrHistory[client.GetId()] = true
			client.WritePacket(packet)
		}
	}
}

func (p *Processor) ProcessPublish(client ifs.Client, packet *packets.PublishPacket) {
	topic := packet.TopicName
	switch packet.Qos {
	case config.QosAtMostOnce:
		p.DoPublish(topic, packet)

	case config.QosAtLeastOnce:
		puback := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
		puback.MessageID = packet.MessageID
		if err := p.WritePacket(client, puback); err != nil {
			logger.Errorf("send puback error: %s\n", err)
			return
		}
		p.DoPublish(topic, packet)

	case config.QosExactlyOnce:
		if err := p.saveQos2Msg(packet.MessageID); nil != err {
			return
		}
		pubrec := packets.NewControlPacket(packets.Pubrec).(*packets.PubrecPacket)
		pubrec.MessageID = packet.MessageID
		if err := p.WritePacket(client, pubrec); err != nil {
			logger.Errorf("send pubrec error: %s\n", err)
			return
		}
		p.DoPublish(topic, packet)
	default:
		logger.Error("publish with unknown qos")
		return
	}

	if packet.Retain {
		if len(packet.Payload) > 0 {
			if !p.s.BrokerTopics().AddRetain(topic, packet) {
				//if not exist old retain, add retain msg count
				atomic.AddInt64(&p.s.State().Retain, 1)
			}
		} else {
			//if exist old retain, reduce retain msg count
			if p.s.BrokerTopics().RemoveRetain(topic) {
				atomic.AddInt64(&p.s.State().Retain, -1)
			}
		}
	}
}

func (p *Processor) ProcessUnSubscribe(client ifs.Client, packet *packets.UnsubscribePacket) {
	topics := packet.Topics
	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID
	p.WritePacket(client, unsuback)

	for _, topic := range topics {
		client.RemoveTopic(topic)
		if p.s.BrokerTopics().Unsubscribe(topic, client.GetId()) {
			// if the subcribe topic is exist, reduce the count
			atomic.AddInt64(&p.s.State().SubCount, -1)
		}
	}

	p.s.Clusters().Range(func(k, v interface{}) bool {
		v.(ifs.Client).WritePacket(packet)
		return true
	})
}

func (p *Processor) ProcessSubscribe(client ifs.Client, packet *packets.SubscribePacket) {
	topics := packet.Topics
	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	p.WritePacket(client, suback)

	for _, topic := range topics {
		client.AddTopic(topic, client.GetId())
		if !p.s.BrokerTopics().Subscribe(topic, client.GetId(), client) {
			//if subcribe topic not exist, add subcribe count
			atomic.AddInt64(&p.s.State().SubCount, 1)
		}
		retains, _ := p.s.BrokerTopics().SearchRetain(topic)
		for _, retain := range retains {
			pubpack := retain.(*packets.PublishPacket)
			p.WritePacket(client, pubpack)
		}
	}

	p.s.Clusters().Range(func(k, v interface{}) bool {
		v.(ifs.Client).WritePacket(packet)
		return true
	})
}

func (p *Processor) ProcessPing(client ifs.Client) {
	pong := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	if err := p.WritePacket(client, pong); err != nil {
		return
	}
}

func (p *Processor) ProcessDisconnect(client ifs.Client) {
	logger.Debugf("broker ProcessDisconnect clientId:%s\n", client.GetId())
	client.Close()

	for topic := range client.Topics() {
		logger.Debugf("ProcessDisconnect topic:%s", topic)
		client.RemoveTopic(topic)
		if p.s.BrokerTopics().Unsubscribe(topic, client.GetId()) {
			// if the subcribe topic is exist, reduce the count
			logger.Debugf("ProcessDisconnect exist topic:%s", topic)
			atomic.AddInt64(&p.s.State().SubCount, -1)
		}
	}

	atomic.AddInt64(&p.s.State().ClientsConnected, -1)
	atomic.AddInt64(&p.s.State().ClientsDisconnected, 1)
}

func (p *Processor) ProcessPubrel(client ifs.Client, cp packets.ControlPacket) {
	packet := cp.(*packets.PubrelPacket)
	p.checkQos2Msg(packet.MessageID)
	pubcomp := packets.NewControlPacket(packets.Pubcomp).(*packets.PubcompPacket)
	pubcomp.MessageID = packet.MessageID
	if err := p.WritePacket(client, pubcomp); err != nil {
		logger.Debugf("send pubcomp error: %s\n", err)
		return
	}
}

func (p *Processor) ProcessConnack(client ifs.Client, cp *packets.ConnackPacket) {
}

func (p *Processor) ProcessConnect(client ifs.Client, cp *packets.ConnectPacket) {
	clientId := cp.ClientIdentifier
	if old, ok := p.s.Clients().Load(clientId); ok {
		oldClient := old.(ifs.Client)
		oldClient.Close()
	}

	connack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	connack.SessionPresent = cp.CleanSession
	connack.ReturnCode = cp.Validate()

	err := p.WritePacket(client, connack)
	if err != nil {
		logger.Error("send connack error, ", err)
		return
	}

	client.SetId(clientId)
	p.s.Clients().Store(clientId, client)

	if cp.WillFlag {
		will := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
		will.Retain = cp.WillRetain
		will.Payload = cp.WillMessage
		will.TopicName = cp.WillTopic
		will.Qos = cp.WillQos
		will.Dup = cp.Dup
		client.(*Client).SetWill(will)
	}

	atomic.AddInt64(&p.s.State().ClientsTotal, 1)
	atomic.AddInt64(&p.s.State().ClientsConnected, 1)
	if atomic.LoadInt64(&p.s.State().ClientsConnected) > atomic.LoadInt64(&p.s.State().ClientsMax) {
		atomic.AddInt64(&p.s.State().ClientsMax, 1)
	}
}

func (p *Processor) WritePacket(client ifs.Client, packet packets.ControlPacket) error {
	err := client.WritePacket(packet)
	if err != nil {
		logger.Error("write packet error, ", err)
		return err
	}
	atomic.AddInt64(&p.s.State().MsgSent, 1)
	atomic.AddInt64(&p.s.State().BytesSent, int64(packet.Size()))
	return nil
}

func (p *Processor) ProcessMessage(client ifs.Client, cp packets.ControlPacket) {

	atomic.AddInt64(&p.s.State().BytesRecv, int64(cp.Size()))
	atomic.AddInt64(&p.s.State().MsgRecv, 1)
	switch packet := cp.(type) {
	case *packets.PublishPacket:
		p.ProcessPublish(client, packet)
	case *packets.PubrelPacket:
		p.ProcessPubrel(client, packet)
	case *packets.SubscribePacket:
		p.ProcessSubscribe(client, packet)
	case *packets.UnsubscribePacket:
		p.ProcessUnSubscribe(client, packet)
	case *packets.PingreqPacket:
		p.ProcessPing(client)
	case *packets.DisconnectPacket:
		p.ProcessDisconnect(client)
	case *packets.ConnectPacket:
		p.ProcessConnect(client, packet)

	case *packets.PubcompPacket:
	case *packets.SubackPacket:
	case *packets.UnsubackPacket:
	case *packets.PingrespPacket:
	case *packets.PubackPacket:
	case *packets.PubrecPacket:
	case *packets.ConnackPacket:
	default:
		logger.Error("Recv Unknow message")
	}
}
