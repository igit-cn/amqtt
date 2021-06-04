package broker

import (
	"errors"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/clients"
	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/ifs"
	"time"
)

const (
	Qos2Timeout int64 = 20
)

type Processor struct {
	server ifs.Server
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
		fmt.Printf("Dropped qos2 packet for too many awaiting_rel messageId:%d\n", messageId)
		return errors.New("RC_PACKET_IDENTIFIER_NOT_FOUND")
	}
	return nil
}

func (s *Processor) saveQos2Msg(messageId uint16) error {
	if s.isQos2MsgFull() {
		fmt.Printf("Dropped qos2 packet for too many awaiting_rel")
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
			fmt.Printf("Dropped qos2 packet for await_rel_timeout messageId:%d\n", messageId)
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
	brokerSubs := s.server.BrokerTopics().Subscribers(topic)
	for _, sub := range brokerSubs {
		if sub != nil {
			sub.(*clients.Client).WritePacket(packet)
		}
	}

	if packet.Retain {
		s.server.ClusterClients().Range(func(key, value interface{}) bool {
			c := value.(*clients.Client)
			c.WritePacket(packet)
			return true
		})
	} else {
		clusterSubs := s.server.ClusterTopics().Subscribers(topic)
		for _, sub := range clusterSubs {
			if sub != nil {
				sub.(*clients.Client).WritePacket(packet)
			}
		}
	}
}

func (s *Processor) ProcessPublish(client *clients.Client, packet *packets.PublishPacket) {
	topic := packet.TopicName
	switch packet.Qos {
	case config.QosAtMostOnce:
		s.DoPublish(topic, packet)

	case config.QosAtLeastOnce:
		puback := packets.NewControlPacket(packets.Puback).(*packets.PubackPacket)
		puback.MessageID = packet.MessageID
		if err := client.WritePacket(puback); err != nil {
			fmt.Printf("send puback error: %s\n", err)
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
			fmt.Printf("send pubrec error: %s\n", err)
			return
		}
		s.DoPublish(topic, packet)
	default:
		fmt.Printf("publish with unknown qos")
		return
	}

	if packet.Retain {
		if len(packet.Payload) > 0 {
			retain := clients.NewRetain(packet)
			s.server.BrokerTopics().AddRetain(topic, retain)
		} else {
			s.server.BrokerTopics().RemoveRetain(topic)
		}
	}
}

func (s *Processor) ProcessUnSubscribe(client *clients.Client, packet *packets.UnsubscribePacket) {
	topics := packet.Topics
	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID

	for _, topic := range topics {
		s.server.BrokerTopics().Unsubscribe(topic, client.GetId())
	}
	client.WritePacket(unsuback)
}

func (s *Processor) ProcessSubscribe(client *clients.Client, packet *packets.SubscribePacket) {
	topics := packet.Topics
	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	client.WritePacket(suback)

	for _, topic := range topics {
		s.server.BrokerTopics().Subscribe(topic, client.GetId(), client)
		retains, _ := s.server.BrokerTopics().SearchRetain(topic)
		for _, retain := range retains {
			client.WritePacket(retain.(*clients.Retain).Packet)
		}
	}

	s.server.ClusterClients().Range(func(k, v interface{}) bool{
		v.(*clients.Client).WritePacket(packet)
		return true
	})
}

func (s *Processor) ProcessPing(client *clients.Client) {
	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	err := client.WritePacket(resp)
	if err != nil {
		fmt.Printf("send PingResponse error: %s\n", err)
		return
	}
}

func (s *Processor) ProcessDisconnect(client *clients.Client) {
	client.Close()
}

func (s *Processor) ProcessPubrel(client *clients.Client, cp packets.ControlPacket) {
	packet := cp.(*packets.PubrelPacket)
	fmt.Printf("ProcessPubrel clientId:%s, MessageID:%d\n", client.GetId(), packet.MessageID)
	s.checkQos2Msg(packet.MessageID)
	pubcomp := packets.NewControlPacket(packets.Pubcomp).(*packets.PubcompPacket)
	pubcomp.MessageID = packet.MessageID
	if err := client.WritePacket(pubcomp); err != nil {
		fmt.Printf("send pubcomp error: %s\n", err)
		return
	}
}

func (s *Processor) ProcessConnect(client *clients.Client, cp *packets.ConnectPacket) {
	clientId := cp.ClientIdentifier
	fmt.Printf("clientId:%s\n", clientId)

	if old, ok := s.server.BrokerClients().Load(clientId); ok {
		oldClient := old.(*clients.Client)
		oldClient.Close()
	}

	connack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	connack.SessionPresent = cp.CleanSession
	connack.ReturnCode = cp.Validate()

	err := client.WritePacket(connack)
	if err != nil {
		fmt.Println("send connack error, ", err)
		return
	}
	client.SetId(clientId)
	s.server.BrokerClients().Store(clientId, client)
}

func (s *Processor) ProcessMessage(client *clients.Client, cp packets.ControlPacket) {
	switch packet := cp.(type) {
	case *packets.PublishPacket:
		s.ProcessPublish(client, packet)
	case *packets.PubrelPacket:
		fmt.Printf("PubrelPacket clientId:%s\n", client.GetId())
		s.ProcessPubrel(client, packet)
	case *packets.SubscribePacket:
		s.ProcessSubscribe(client, packet)
	case *packets.UnsubscribePacket:
		fmt.Printf("UnsubscribePacket clientId:%s\n", client.GetId())
		s.ProcessUnSubscribe(client, packet)
	case *packets.PingreqPacket:
		fmt.Printf("PingreqPacket clientId:%s\n", client.GetId())
		s.ProcessPing(client)
	case *packets.DisconnectPacket:
		fmt.Printf("DisconnectPacket clientId:%s\n", client.GetId())
		s.ProcessDisconnect(client)

	case *packets.PubcompPacket:
	case *packets.SubackPacket:
	case *packets.UnsubackPacket:
	case *packets.PingrespPacket:
	case *packets.PubackPacket:
	case *packets.PubrecPacket:
	case *packets.ConnackPacket:
	case *packets.ConnectPacket:
	default:
		fmt.Printf("Recv Unknow message")
	}
}