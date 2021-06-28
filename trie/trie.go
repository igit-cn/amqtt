package trie

import (
	"strings"
	"sync"
)

func NewTrie() *Trie {
	trie := new(Trie)
	trie.root = NewBranch(trie)
	trie.root.SetName("root")
	return trie
}

type Trie struct {
	mu      sync.RWMutex
	root    *Branch
	foliage sync.Map
}

func (t *Trie) Subscribe(topic string, identity string, subscriber interface{}) error {
	if strings.Contains(topic, "#") || strings.Contains(topic, "+") {
		t.mu.Lock()
		defer t.mu.Unlock()
		keys := strings.Split(topic, "/")
		index := 0
		if strings.TrimSpace(keys[0]) == "" {
			index = 1
		}
		t.root.AddBranch(topic, identity, subscriber, keys, index)
	} else {
		leaf, ok := t.foliage.Load(topic)
		if !ok {
			leaf = NewLeaf(nil)
			t.foliage.Store(topic, leaf)
		}
		leaf.(*Leaf).AddSubscriber(identity, subscriber)
	}
	return nil
}

func (t *Trie) Unsubscribe(topic string, identity string) error {
	_, ok := t.foliage.Load(topic)
	if ok {
		t.foliage.Delete(topic)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}

	for _, branch := range t.root.GetBranches() {
		branch.ScanRemoveLeaf(identity, topic, keys, index)
	}
	return nil
}

func (t *Trie) Subscribers(topic string) []interface{} {
	subscribers := make([]interface{}, 0)
	leaf, ok := t.foliage.Load(topic)
	if ok {
		for _, subscriber := range leaf.(*Leaf).Subscribers() {
			subscribers = append(subscribers, subscriber)
		}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}

	for _, branch := range t.root.GetBranches() {
		leaves := branch.SearchLeaves(topic, keys, index)
		for _, leaf := range leaves {
			for _, subscriber := range leaf.Subscribers() {
				subscribers = append(subscribers, subscriber)
			}
		}
	}
	return subscribers
}

func (t *Trie) AddRetain(topic string, packet interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	t.root.AddRetain(topic, keys, index, packet)
	return nil
}

func (t *Trie) RemoveRetain(topic string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	for _, branch := range t.root.GetBranches() {
		branch.RemoveRetain(topic, keys, index)
	}
	return nil
}

func (t *Trie) SearchRetain(topic string) ([]interface{}, error) {
	retains := make([]interface{}, 0)
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	for _, branch := range t.root.GetBranches() {
		retains = append(retains, branch.SearchRetain(topic, keys, index)...)
	}
	return retains, nil
}
