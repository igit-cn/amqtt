package server

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/clients"
	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/topics"
)

const (
	Qos2Timeout int64 = 20
)

type Server struct {
	Clients sync.Map
	Topics  *topics.Trie

	qos2Msgs    map[uint16]int64
	maxQos2Msgs int
}

func NewServer() *Server {
	s := new(Server)
	s.Topics = topics.NewTrie()
	s.qos2Msgs = make(map[uint16]int64)
	return s
}

func (s *Server) checkQos2Msg(messageId uint16) error {
	if _, found := s.qos2Msgs[messageId]; found {
		delete(s.qos2Msgs, messageId)
	} else {
		fmt.Printf("Dropped qos2 packet for too many awaiting_rel messageId:%d\n", messageId)
		return errors.New("RC_PACKET_IDENTIFIER_NOT_FOUND")
	}
	return nil
}

func (s *Server) saveQos2Msg(messageId uint16) error {
	if s.isQos2MsgsFull() {
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

func (s *Server) expireQos2Msg() {
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

func (s *Server) isQos2MsgsFull() bool {
	if s.maxQos2Msgs == 0 {
		return false
	}
	if len(s.qos2Msgs) < s.maxQos2Msgs {
		return false
	}
	return true
}

func (s *Server) DoPublish(topic string, packet *packets.PublishPacket) {
	leaves := s.Topics.Search(topic)
	for _, leaf := range leaves {
		if leaf != nil {
			for _, client := range leaf.Clients {
				client.WritePacket(packet)
			}
		}
	}
}

func (s *Server) ProcessPublish(client *clients.Client, packet *packets.PublishPacket) {
	topic := packet.TopicName
	// fmt.Printf("ProcessPublish topic:%+v, clientId:%s, MessageID:%d\n", topic, client.ClientId, packet.MessageID)

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
			retain := topics.NewRetain(packet)
			s.Topics.AddRetain(topic, retain)
		} else {
			s.Topics.RemoveRetain(topic)
		}
	}
}

func (s *Server) ProcessUnSubscribe(client *clients.Client, packet *packets.UnsubscribePacket) {
	topics := packet.Topics
	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID

	for _, topic := range topics {
		s.Topics.Remove(client, topic)
	}
	client.WritePacket(unsuback)
}

func (s *Server) ProcessSubscribe(client *clients.Client, packet *packets.SubscribePacket) {
	topics := packet.Topics
	fmt.Printf("ProcessSubscribe topics:%+v, clientId:%s\n", topics, client.ClientId)
	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	client.WritePacket(suback)

	for _, topic := range topics {
		s.Topics.Parse(client, topic)
		retains := s.Topics.SearchRetain(topic)
		fmt.Printf("ProcessSubscribe retains:%+v\n", retains)
		for _, retain := range retains {
			client.WritePacket(retain.Packet)
		}
	}
}

func (s *Server) ProcessPing(client *clients.Client) {
	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	err := client.WritePacket(resp)
	if err != nil {
		fmt.Printf("send PingResponse error: %s\n", err)
		return
	}
}

func (s *Server) ProcessDisconnect(client *clients.Client) {
	s.RemoveClientSubs(client)
	client.CancelFunc()
	client.Close()
}

func (s *Server) ProcessPubrel(client *clients.Client, cp packets.ControlPacket) {
	packet := cp.(*packets.PubrelPacket)
	fmt.Printf("ProcessPubrel clientId:%s, MessageID:%d\n", client.ClientId, packet.MessageID)
	s.checkQos2Msg(packet.MessageID)
	pubcomp := packets.NewControlPacket(packets.Pubcomp).(*packets.PubcompPacket)
	pubcomp.MessageID = packet.MessageID
	if err := client.WritePacket(pubcomp); err != nil {
		fmt.Printf("send pubcomp error: %s\n", err)
		return
	}
}

func (s *Server) ProcessMessage(client *clients.Client, cp packets.ControlPacket) {
	switch packet := cp.(type) {
	case *packets.ConnackPacket:
	case *packets.ConnectPacket:
	case *packets.PublishPacket:
		s.ProcessPublish(client, packet)

	case *packets.PubackPacket:
		fmt.Printf("PubackPacket clientId:%s\n", client.ClientId)

	case *packets.PubrecPacket:
		fmt.Printf("PubrecPacket clientId:%s\n", client.ClientId)

	case *packets.PubrelPacket:
		fmt.Printf("PubrelPacket clientId:%s\n", client.ClientId)
		s.ProcessPubrel(client, packet)

	case *packets.PubcompPacket:
		fmt.Printf("PubcompPacket clientId:%s\n", client.ClientId)

	case *packets.SubscribePacket:
		s.ProcessSubscribe(client, packet)

	case *packets.SubackPacket:
		fmt.Printf("SubackPacket clientId:%s\n", client.ClientId)

	case *packets.UnsubscribePacket:
		fmt.Printf("UnsubscribePacket clientId:%s\n", client.ClientId)
		s.ProcessUnSubscribe(client, packet)

	case *packets.UnsubackPacket:
		fmt.Printf("UnsubackPacket clientId:%s\n", client.ClientId)

	case *packets.PingreqPacket:
		fmt.Printf("PingreqPacket clientId:%s\n", client.ClientId)
		s.ProcessPing(client)

	case *packets.PingrespPacket:
		fmt.Printf("PingreqPacket clientId:%s\n", client.ClientId)

	case *packets.DisconnectPacket:
		fmt.Printf("DisconnectPacket clientId:%s\n", client.ClientId)
		s.ProcessDisconnect(client)

	default:
		fmt.Printf("Recv Unknow message")
	}
}

func (s *Server) ReadLoop(client *clients.Client) {
	for {
		select {
		case <-client.Ctx.Done():
			return
		default:
			packet, err := client.ReadPacket()
			if err != nil {
				fmt.Printf("read packet error: %+v\n", err)
				packet := packets.NewControlPacket(packets.Disconnect).(*packets.DisconnectPacket)
				s.ProcessMessage(client, packet)
				return
			}
			s.ProcessMessage(client, packet)
		}
	}
}

func (s *Server) RemoveClientSubs(client *clients.Client) {
	for _, topic := range client.Subs {
		s.Topics.Remove(client, topic)
	}
}

func (s *Server) Handler(conn net.Conn) {
	client := clients.NewClient(conn)
	packet, err := client.ReadPacket()

	if err != nil {
		fmt.Println("read connect packet error: ", err)
		return
	}
	if packet == nil {
		fmt.Println("received nil packet")
		return
	}

	msg, ok := packet.(*packets.ConnectPacket)
	if !ok {
		fmt.Println("received msg that was not Connect")
		return
	}

	clientId := msg.ClientIdentifier
	fmt.Printf("clientId:%s\n", clientId)

	if old, ok := s.Clients.Load(clientId); ok {
		oldClient := old.(*clients.Client)
		s.RemoveClientSubs(oldClient)
		oldClient.CancelFunc()
		oldClient.Close()
	}

	connack := packets.NewControlPacket(packets.Connack).(*packets.ConnackPacket)
	connack.SessionPresent = msg.CleanSession
	connack.ReturnCode = msg.Validate()

	err = connack.Write(conn)
	if err != nil {
		fmt.Println("send connack error, ", err)
		return
	}

	client.SetClientId(clientId)
	s.Clients.Store(clientId, client)
	s.ReadLoop(client)
}
