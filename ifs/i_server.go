package ifs

import (
	"context"
	"sync"
	"time"
)

type ServerState struct {
	//The total number of bytes received since the broker started.
	//$SYS/broker/bytes/received
	BytesRecv int64 `json:"bytes_recv"`

	//The total number of bytes sent since the broker started.
	//$SYS/broker/bytes/sent
	BytesSent int64 `json:"bytes_sent"`

	//The number of currently connected clients.
	//$SYS/broker/clients/connected
	ClientsConnected int64 `json:"clients_connected"`

	//The total number of persistent clients (with clean session disabled) that are registered at the broker but are currently disconnected
	//$SYS/broker/clients/disconnected
	ClientsDisconnected int64 `json:"clients_disconnected"`

	//The maximum number of clients that have been connected to the broker at the same time.
	//$SYS/broker/clients/maximum
	ClientsMax int64 `json:"clients_max"`

	//The total number of active and inactive clients currently connected and registered on the broker.
	//$SYS/broker/clients/total
	ClientsTotal int64 `json:"clients_total"`

	//The number of messages with QoS>0 that are awaiting acknowledgments.
	//$SYS/broker/messages/inflight
	Inflight int64 `json:"inflight"`

	//The total number of messages of any type received since the broker started.
	//$SYS/broker/messages/received
	MsgRecv int64 `json:"msg_recv"`

	//The total number of messages of any type sent since the broker started.
	//$SYS/broker/messages/sent
	MsgSent int64 `json:"msg_sent"`

	//The total number of PUBLISH messages received since the broker started.
	//$SYS/broker/publish/messages/received
	PubRecv int64 `json:"pub_recv"`

	//The total number of PUBLISH messages sent since the broker started.
	//$SYS/broker/publish/messages/sent
	PubSent int64 `json:"pub_sent"`

	//The total number of retained messages active on the broker.
	//$SYS/broker/retained messages/count
	Retain int64 `json:"retain"`

	//The number of messages currently held in the message store. This includes retained messages and messages queued for durable clients.
	//$SYS/broker/store/messages/count
	StoreCount int64 `json:"store_count"`

	//The number of bytes currently held by message payloads in the message store. This includes retained messages and messages queued for durable clients.
	//$SYS/broker/store/messages/bytes
	StoreBytes int64 `json:"storn_bytes"`

	//The total number of subscriptions active on the broker.
	//$SYS/broker/subscriptions/count
	SubCount int64 `json:"sub_count"`

	//The version of the broker. Static.
	//$SYS/broker/version
	Version string `json:"version"`

	//"$SYS/broker/uptime":
	Uptime int64 `json:"uptime"`
}

func NewState(version string) *ServerState {
	state := new(ServerState)
	state.Version = "1.0.0"
	state.Uptime = time.Now().Unix()
	return state
}

type Server interface {
	Context() context.Context
	BrokerTopics() TopicStorage
	ClusterTopics() TopicStorage
	Clients() *sync.Map
	Clusters() *sync.Map
	State() *ServerState
	Close()
}
