package server

import (
	"context"
	"sync"

	"github.com/werbenhu/amq/broker"
	"github.com/werbenhu/amq/cluster"
	"github.com/werbenhu/amq/config"
	"github.com/werbenhu/amq/ifs"
	"github.com/werbenhu/amq/logger"
	"github.com/werbenhu/amq/trie"
)

type Server struct {
	ctx           context.Context
	cancel        context.CancelFunc
	brokerTopics  ifs.Topic
	clusterTopics ifs.Topic
	clients       *sync.Map
	clusters      *sync.Map
	b             *broker.Broker
	c             *cluster.Cluster
}

func NewServer(ctx context.Context) *Server {
	s := new(Server)
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.brokerTopics = trie.NewTrie()
	s.clusterTopics = trie.NewTrie()
	s.clients = new(sync.Map)
	s.clusters = new(sync.Map)
	s.b = broker.NewBroker(s)
	s.c = cluster.NewCluster(s)
	return s
}

func (s *Server) Start() {
	go s.b.StartTcp()
	go s.b.StartWebsocket()

	if config.IsCluster() {
		go s.c.Start()
	}

	select {
	case <-s.ctx.Done():
		logger.Debugf("server done")
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

func (s *Server) Clients() *sync.Map {
	return s.clients
}

func (s *Server) Clusters() *sync.Map {
	return s.clusters
}
