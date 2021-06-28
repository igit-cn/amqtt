package broker

import (
	"errors"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/ifs"
	"github.com/werbenhu/amq/logger"
)

const (
	Qos2Timeout int64 = 20
)

type Processor struct {
	server      ifs.Server
	qos2Msgs    map[uint16]int64
	maxQos2Msgs int
}

func NewProcessor(server ifs.Server) *Processor {
	p := new(Processor)
	p.server = server
	p.qos2Msgs = make(map[uint16]int64)
	return p
}

func (s *Processor) checkQos2Msg(messageId uint16) error {
	if _, found := s.qos2Msgs[messageId]; found {
		delete(s.qos2Msgs, messageId)
	} else {
		return errors.New("RC_PACKET_IDENTIFIER_NOT_FOUND")
	}
	return nil
}

func (s *Processor) saveQos2Msg(messageId uint16) error {
	if s.isQos2MsgFull() {
		return errors.New("DROPPED_QOS2_PACKET_FOR_TOO_MANY_AWAITING_REL")
	}

	if _, found := s.qos2Msgs[messageId]; found {
		return errors.New("RC_PACKET_IDENTIFIER_IN_USE")
	}
	s.qos2Msgs[messageId] = time.Now().Unix()
	time.AfterFunc(time.Duration(Qos2Timeout)*time.Second, s.expireQos2Msg)
	return nil
}

func (s *Processor) expireQos2Msg() {
	if len(s.qos2Msgs) == 0 {
		return
	}
	now := time.Now().Unix()
	for messageId, ts := range s.qos2Msgs {
		if now-ts >= Qos2Timeout {
			delete(s.qos2Msgs, messageId)
		}
	}
}

func (s *Processor) isQos2MsgFull() bool {
	if s.maxQos2Msgs == 0 {
		return false
	}
	if len(s.qos2Msgs) < s.maxQos2Msgs {
		return false
	}
	return true
}

func (s *Processor) DoPublish(topic string, packet *packets.PublishPacket) {
	logger.Debugf("DoPublish topic:%s, packet:%+v", topic, packet)
	brokerSubs := s.server.BrokerTopics().Subscribers(topic)
	for _, sub := range brokerSubs {
		if sub != nil {
			logger.Debugf("DoPublish sub c:%s", sub.(ifs.Client).GetId())
			sub.(ifs.Client).WritePacket(packet)
		}
	}

	clusterSubs := s.server.ClusterTopics().Subscribers(topic)
	for _, sub := range clusterSubs {
		if sub != nil {
			sub.(ifs.Client).WritePacket(packet)
		}
	}
}

func (s *Processor) ProcessPublish(client ifs.Client, packet *packets.PublishPacket) {
	topic := packet.TopicName
	switch packet.Qos {
	case config.QosAtMostOnce:
		s.DoPublish(topic, packet)

	case config.QosAtLeastOnce:
		puback := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
		puback.MessageID = packet.MessageID
		if err := client.WritePacket(puback); err != nil {
			logger.Errorf("send puback error: %s\n", err)
			return
		}
		s.DoPublish(topic, packet)

	case config.QosExactlyOnce:
		if err := s.saveQos2Msg(packet.MessageID); nil != err {
			return
		}
		pubrec := packets.NewControlPacket(packets.Pubrec).(*packets.PubrecPacket)
		pubrec.MessageID = packet.MessageID
		if err := client.WritePacket(pubrec); err != nil {
			logger.Errorf("send pubrec error: %s\n", err)
			return
		}
		s.DoPublish(topic, packet)
	default:
		logger.Error("publish with unknown qos")
		return
	}

	if packet.Retain {
		if len(packet.Payload) > 0 {
			s.server.BrokerTopics().AddRetain(topic, packet)
		} else {
			s.server.BrokerTopics().RemoveRetain(topic)
		}
	}
}

func (s *Processor) ProcessUnSubscribe(client ifs.Client, packet *packets.UnsubscribePacket) {
	topics := packet.Topics
	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID

	for _, topic := range topics {
		s.server.BrokerTopics().Unsubscribe(topic, client.GetId())
	}
	client.WritePacket(unsuback)
}

func (s *Processor) ProcessSubscribe(client ifs.Client, packet *packets.SubscribePacket) {
	topics := packet.Topics
	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	client.WritePacket(suback)

	for _, topic := range topics {
		s.server.BrokerTopics().Subscribe(topic, client.GetId(), client)
		retains, _ := s.server.BrokerTopics().SearchRetain(topic)
		for _, retain := range retains {
			client.WritePacket(retain.(*packets.PublishPacket))
		}
	}

	s.server.Clusters().Range(func(k, v interface{}) bool {
		v.(ifs.Client).WritePacket(packet)
		return true
	})
}

func (s *Processor) ProcessPing(client ifs.Client) {
	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	err := client.WritePacket(resp)
	if err != nil {
		return
	}
}

func (s *Processor) ProcessDisconnect(client ifs.Client) {
	logger.Debugf("broker ProcessDisconnect clientId:%s\n", client.GetId())
	client.Close()
}

func (s *Processor) ProcessPubrel(client ifs.Client, cp packets.ControlPacket) {
	packet := cp.(*packets.PubrelPacket)
	s.checkQos2Msg(packet.MessageID)
	pubcomp := packets.NewControlPacket(packets.Pubcomp).(*packets.PubcompPacket)
	pubcomp.MessageID = packet.MessageID
	if err := client.WritePacket(pubcomp); err != nil {
		logger.Debugf("send pubcomp error: %s\n", err)
		return
	}
}

func (s *Processor) ProcessConnack(client ifs.Client, cp *packets.ConnackPacket) {
}

func (s *Processor) ProcessConnect(client ifs.Client, cp *packets.ConnectPacket) {
	clientId := cp.ClientIdentifier
	if old, ok := s.server.Clients().Load(clientId); ok {
		oldClient := old.(ifs.Client)
		oldClient.Close()
	}

	connack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	connack.SessionPresent = cp.CleanSession
	connack.ReturnCode = cp.Validate()

	err := client.WritePacket(connack)
	if err != nil {
		logger.Error("send connack error, ", err)
		return
	}

	client.SetId(clientId)
	s.server.Clients().Store(clientId, client)

	if cp.WillFlag {
		will := packets.NewControlPacket(packets.Publish).(*packets.PublishPacket)
		will.Retain = cp.WillRetain
		will.Payload = cp.WillMessage
		will.TopicName = cp.WillTopic
		will.Qos = cp.WillQos
		will.Dup = cp.Dup
		client.(*Client).SetWill(will)
	}
}

func (s *Processor) ProcessMessage(client ifs.Client, cp packets.ControlPacket) {
	switch packet := cp.(type) {
	case *packets.PublishPacket:
		s.ProcessPublish(client, packet)
	case *packets.PubrelPacket:
		s.ProcessPubrel(client, packet)
	case *packets.SubscribePacket:
		s.ProcessSubscribe(client, packet)
	case *packets.UnsubscribePacket:
		s.ProcessUnSubscribe(client, packet)
	case *packets.PingreqPacket:
		s.ProcessPing(client)
	case *packets.DisconnectPacket:
		s.ProcessDisconnect(client)
	case *packets.ConnectPacket:
		s.ProcessConnect(client, packet)

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
