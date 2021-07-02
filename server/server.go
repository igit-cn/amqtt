package server

import (
	"context"
	"sync"

	"github.com/werbenhu/amqtt/broker"
	"github.com/werbenhu/amqtt/cluster"
	"github.com/werbenhu/amqtt/config"
	"github.com/werbenhu/amqtt/ifs"
	"github.com/werbenhu/amqtt/logger"
	"github.com/werbenhu/amqtt/trie"
)

type Server struct {
	ctx           context.Context
	cancel        context.CancelFunc
	brokerTopics  ifs.TopicStorage
	clusterTopics ifs.TopicStorage
	clients       *sync.Map
	clusters      *sync.Map
	b             *broker.Broker
	c             *cluster.Cluster
	state         *ifs.ServerState
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
	s.state = ifs.NewState("1.0.0")
	return s
}

func (s *Server) Start() {
	go s.b.StartTcp()

	if config.IsWebsocket() {
		go s.b.StartWebsocket()
	}

	if config.IsCluster() {
		go s.c.Start()
	}

	<-s.ctx.Done()
	logger.Debugf("server done")
}

func (s *Server) Close() {
	s.cancel()
}

func (s *Server) Context() context.Context {
	return s.ctx
}

func (s *Server) BrokerTopics() ifs.TopicStorage {
	return s.brokerTopics
}

func (s *Server) ClusterTopics() ifs.TopicStorage {
	return s.clusterTopics
}

func (s *Server) Clients() *sync.Map {
	return s.clients
}

func (s *Server) Clusters() *sync.Map {
	return s.clusters
}

func (s *Server) State() *ifs.ServerState {
	return s.state
}

func (s *Server) AddSub() {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	return s.subs
}
