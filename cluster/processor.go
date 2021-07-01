package cluster

import (
	"github.com/werbenhu/amqtt/ifs"
	"github.com/werbenhu/amqtt/logger"
	"github.com/werbenhu/amqtt/packets"
)

type Processor struct {
	s ifs.Server
}

func NewProcessor(server ifs.Server) *Processor {
	p := new(Processor)
	p.s = server
	return p
}

func (p *Processor) DoPublish(topic string, packet *packets.PublishPacket) {
	subs := p.s.BrokerTopics().Subscribers(topic)
	for _, sub := range subs {
		if sub != nil {
			sub.(ifs.Client).WritePacket(packet)
		}
	}
}

func (p *Processor) ProcessPublish(client ifs.Client, packet *packets.PublishPacket) {
	topic := packet.TopicName
	p.DoPublish(topic, packet)

	//If other cluster node have this retain message, then the current cluster node must delete this retain message
	//Ensure that there is only one retain message for a topic in the entire clusters
	if packet.Retain {
		p.s.BrokerTopics().RemoveRetain(topic)
	}
}

func (p *Processor) ProcessUnSubscribe(client ifs.Client, packet *packets.UnsubscribePacket) {
	topics := packet.Topics
	unsuback := packets.NewControlPacket(packets.Unsuback).(*packets.UnsubackPacket)
	unsuback.MessageID = packet.MessageID

	for _, topic := range topics {
		p.s.ClusterTopics().Unsubscribe(topic, client.GetId())
		client.RemoveTopic(topic)
	}
	client.WritePacket(unsuback)
}

func (p *Processor) ProcessSubscribe(client ifs.Client, packet *packets.SubscribePacket) {
	topics := packet.Topics
	suback := packets.NewControlPacket(packets.Suback).(*packets.SubackPacket)
	suback.MessageID = packet.MessageID
	client.WritePacket(suback)

	for _, topic := range topics {
		p.s.ClusterTopics().Subscribe(topic, client.GetId(), client)
		client.AddTopic(topic, client.GetId())

		retains, _ := p.s.BrokerTopics().SearchRetain(topic)
		for _, retain := range retains {
			client.WritePacket(retain.(*packets.PublishPacket))
		}
	}
}

func (p *Processor) ProcessPing(client ifs.Client) {
	resp := packets.NewControlPacket(packets.Pingresp).(*packets.PingrespPacket)
	err := client.WritePacket(resp)
	if err != nil {
		logger.Errorf("send PingResponse error: %s\n", err)
		return
	}
}

func (p *Processor) ProcessDisconnect(client ifs.Client) {
	logger.Debugf("cluster ProcessDisconnect clientId:%s", client.GetId())
	p.s.Clusters().Delete(client.GetId())
	client.Close()

	for topic := range client.Topics() {
		logger.Debugf("ProcessDisconnect topic:%s", topic)
		client.RemoveTopic(topic)
		p.s.ClusterTopics().Unsubscribe(topic, client.GetId())
	}
}

func (p *Processor) ProcessConnack(client ifs.Client, cp *packets.ConnackPacket) {
	logger.Debugf("cluster ProcessConnack clientId:%s", client.GetId())
}

func (p *Processor) ProcessConnect(client ifs.Client, cp *packets.ConnectPacket) {
	clientId := cp.ClientIdentifier
	if old, ok := p.s.Clusters().Load(clientId); ok {
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
	p.s.Clusters().Store(clientId, client)
}

func (p *Processor) ProcessMessage(client ifs.Client, cp packets.ControlPacket) {
	switch packet := cp.(type) {
	case *packets.ConnackPacket:
		p.ProcessConnack(client, packet)
	case *packets.ConnectPacket:
		p.ProcessConnect(client, packet)
	case *packets.PublishPacket:
		p.ProcessPublish(client, packet)
	case *packets.SubscribePacket:
		p.ProcessSubscribe(client, packet)
	case *packets.UnsubscribePacket:
		p.ProcessUnSubscribe(client, packet)
	case *packets.PingreqPacket:
		p.ProcessPing(client)
	case *packets.DisconnectPacket:
		p.ProcessDisconnect(client)

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
