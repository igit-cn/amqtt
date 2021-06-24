package ifs

import (
	"context"
	"sync"
)

type Server interface {
	Context() context.Context
	BrokerTopics() Topic
	ClusterTopics() Topic
	Clients() *sync.Map
	Clusters() *sync.Map
}
