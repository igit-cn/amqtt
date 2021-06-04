package server

import (
	"context"
	"fmt"
	"github.com/werbenhu/amq/broker"
	"github.com/werbenhu/amq/cluster"
	"github.com/werbenhu/amq/ifs"
	"github.com/werbenhu/amq/trie"
	"sync"
)

type Server struct {
	ctx context.Context
	cancel context.CancelFunc
	brokerTopics   ifs.Topic
	clusterTopics  ifs.Topic
	brokerClients  *sync.Map
	clusterClients *sync.Map
	b *broker.Broker
	c *cluster.Cluster
}

func NewServer(ctx context.Context) *Server{
	s := new(Server)
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.brokerTopics = trie.NewTrie()
	s.clusterTopics = trie.NewTrie()
	s.brokerClients = new(sync.Map)
	s.clusterClients = new(sync.Map)
	s.b = broker.NewBroker(s)
	s.c = cluster.NewCluster(s)
	return s
}

func (s *Server) Start() {
	go s.b.Start()
	go s.c.Start()

	select {
	case <- s.ctx.Done():
		fmt.Printf("server done")
		return
	}
}

func (s *Server) Context() context.Context {
	return s.ctx
}

func (s *Server) BrokerTopics() ifs.Topic {
	return s.brokerTopics
}

func (s *Server) ClusterTopics() ifs.Topic {
	return s.clusterTopics
}

func (s *Server) BrokerClients() *sync.Map {
	return s.brokerClients
}

func (s *Server) ClusterClients() *sync.Map {
	return s.clusterClients
}