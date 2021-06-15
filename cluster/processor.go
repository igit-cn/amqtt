package cluster

import (
	"github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/werbenhu/amq/ifs"
	"github.com/werbenhu/amq/logger"
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
			sub.(ifs.Client).WritePacket(packet)
		}
	}
}

func (s *Processor) ProcessPublish(client ifs.Client, packet *packets.PublishPacket) {
	topic := packet.TopicName
	s.DoPublish(topic, packet)

	//If other cluster node have this retain message, then the current cluster node must delete this retain message
	//Ensure that there is only one retain message for a topic in the entire clusters
	if packet.Retain {
		s.server.BrokerTopics().RemoveRetain(topic)
	}
}

func (s *Processor) ProcessUnSubscribe(client ifs.Client, packet *packets.UnsubscribePacket) {
	topics := packet.Topics
	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID

	for _, topic := range topics {
		s.server.ClusterTopics().Unsubscribe(topic, client.GetId())
	}
	client.WritePacket(unsuback)
}

func (s *Processor) ProcessSubscribe(client ifs.Client, packet *packets.SubscribePacket) {
	topics := packet.Topics
	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	client.WritePacket(suback)

	for _, topic := range topics {
		s.server.ClusterTopics().Subscribe(topic, client.GetId(), client)
		retains, _ := s.server.BrokerTopics().SearchRetain(topic)
		for _, retain := range retains {
			client.WritePacket(retain.(*packets.PublishPacket))
		}
	}
}

func (s *Processor) ProcessPing(client ifs.Client) {
	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	err := client.WritePacket(resp)
	if err != nil {
		logger.Errorf("send PingResponse error: %s\n", err)
		return
	}
}

func (s *Processor) ProcessDisconnect(client ifs.Client) {
	logger.Debugf("cluster ProcessDisconnect clientId:%s", client.GetId())
	s.server.ClusterClients().Delete(client.GetId())
	client.Close()
}

func (s *Processor) ProcessConnack(client ifs.Client, cp *packets.ConnackPacket) {
	logger.Debugf("cluster ProcessConnack clientId:%s", client.GetId())
}

func (s *Processor) ProcessConnect(client ifs.Client, cp *packets.ConnectPacket) {
	clientId := cp.ClientIdentifier
	if old, ok := s.server.ClusterClients().Load(clientId); ok {
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
	logger.Debugf("cluster ProcessConnect clientId:%s", client.GetId())
	client.SetId(clientId)
	s.server.ClusterClients().Store(clientId, client)
}

func (s *Processor) ProcessMessage(client ifs.Client, cp packets.ControlPacket) {
	switch packet := cp.(type) {
	case *packets.ConnackPacket:
		s.ProcessConnack(client, packet)
	case *packets.ConnectPacket:
		s.ProcessConnect(client, packet)
	case *packets.PublishPacket:
		s.ProcessPublish(client, packet)
	case *packets.SubscribePacket:
		s.ProcessSubscribe(client, packet)
	case *packets.UnsubscribePacket:
		s.ProcessUnSubscribe(client, packet)
	case *packets.PingreqPacket:
		s.ProcessPing(client)
	case *packets.DisconnectPacket:
		s.ProcessDisconnect(client)

	case *packets.PubackPacket:
	case *packets.PubrecPacket:
	case *packets.PubrelPacket:
	case *packets.PubcompPacket:
	case *packets.SubackPacket:
	case *packets.UnsubackPacket:
	case *packets.PingrespPacket:
	default:
		logger.Error("Recv Unknow message")
	}
}
