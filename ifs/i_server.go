package ifs

import (
	"context"
	"sync"
)

type Server interface {
	Context() context.Context
	BrokerTopics() Topic
	ClusterTopics() Topic
	BrokerClients() *sync.Map
	ClusterClients() *sync.Map
}