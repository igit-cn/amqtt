package cluster

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/clients"
	"github.com/werbenhu/amq/ifs"
)

type Processor struct {
	server ifs.Server
}

func NewProcessor(server ifs.Server) *Processor {
	p := new(Processor)
	p.server = server
	return p
}

func (s *Processor) DoPublish(topic string, packet *packets.PublishPacket) {
	subs := s.server.BrokerTopics().Subscribers(topic)
	for _, sub := range subs {
		if sub != nil {
			sub.(*clients.Client).WritePacket(packet)
		}
	}
}

func (s *Processor) ProcessPublish(client *clients.Client, packet *packets.PublishPacket) {
	topic := packet.TopicName
	s.DoPublish(topic, packet)

	//If other cluster node have this retain message, then the current cluster node must delete this retain message
	//Ensure that there is only one retain message for a topic in the entire cluster
	if packet.Retain {
		s.server.BrokerTopics().RemoveRetain(topic)
	}
}

func (s *Processor) ProcessUnSubscribe(client *clients.Client, packet *packets.UnsubscribePacket) {
	topics := packet.Topics
	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID

	for _, topic := range topics {
		s.server.ClusterTopics().Unsubscribe(topic, client.GetId())
	}
	client.WritePacket(unsuback)
}

func (s *Processor) ProcessSubscribe(client *clients.Client, packet *packets.SubscribePacket) {
	topics := packet.Topics
	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	client.WritePacket(suback)

	for _, topic := range topics {
		s.server.ClusterTopics().Subscribe(topic, client.GetId(), client)
		retains, _ := s.server.BrokerTopics().SearchRetain(topic)
		for _, retain := range retains {
			client.WritePacket(retain.(*clients.Retain).Packet)
		}
	}
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

func (s *Processor) ProcessConnack(client *clients.Client, cp *packets.ConnackPacket) {
	clientId := client.GetConn().RemoteAddr().String()
	fmt.Printf("ProcessConnack clientId:%s\n", clientId)

	if old, ok := s.server.ClusterClients().Load(clientId); ok {
		oldClient := old.(*clients.Client)
		oldClient.Close()
	}
	client.SetId(clientId)
	s.server.ClusterClients().Store(clientId, client)
}

func (s *Processor) ProcessConnect(client *clients.Client, cp *packets.ConnectPacket) {

	clientId := cp.ClientIdentifier
	if old, ok := s.server.ClusterClients().Load(clientId); ok {
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
	s.server.ClusterClients().Store(clientId, client)
}

func (s *Processor) ProcessMessage(client *clients.Client, cp packets.ControlPacket) {
	switch packet := cp.(type) {
	case *packets.ConnackPacket:
		fmt.Printf("ConnackPacket clientId:%s\n", client.GetId())
		s.ProcessConnack(client, packet)
	case *packets.ConnectPacket:
		fmt.Printf("ConnectPacket clientId:%s\n", client.GetId())
	case *packets.PublishPacket:
		fmt.Printf("PublishPacket clientId:%s\n", client.GetId())
		s.ProcessPublish(client, packet)
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

	case *packets.PubackPacket:
	case *packets.PubrecPacket:
	case *packets.PubrelPacket:
	case *packets.PubcompPacket:
	case *packets.SubackPacket:
	case *packets.UnsubackPacket:
	case *packets.PingrespPacket:
	default:
		fmt.Printf("Recv Unknow message")
	}
}