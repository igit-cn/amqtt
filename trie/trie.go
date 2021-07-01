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

func (t *Trie) Subscribe(topic string, identity string, subscriber interface{}) (exist bool) {
	if strings.Contains(topic, "#") || strings.Contains(topic, "+") {
		t.mu.Lock()
		defer t.mu.Unlock()
		keys := strings.Split(topic, "/")
		index := 0
		if strings.TrimSpace(keys[0]) == "" {
			index = 1
		}
		return t.root.AddBranch(topic, identity, subscriber, keys, index)
	} else {
		leaf, ok := t.foliage.Load(topic)
		if !ok {
			leaf = NewLeaf(nil)
			t.foliage.Store(topic, leaf)
		}
		return leaf.(*Leaf).AddSubscriber(identity, subscriber)
	}
}

func (t *Trie) Unsubscribe(topic string, identity string) (exist bool) {
	if strings.Contains(topic, "#") || strings.Contains(topic, "+") {
		t.mu.Lock()
		defer t.mu.Unlock()
		keys := strings.Split(topic, "/")
		index := 0
		if strings.TrimSpace(keys[0]) == "" {
			index = 1
		}
		for _, branch := range t.root.GetBranches() {
			if branch.ScanRemoveLeaf(identity, topic, keys, index) {
				exist = true
				break
			}
		}
	} else {
		l, ok := t.foliage.Load(topic)
		if ok {
			leaf := l.(*Leaf)
			leaf.RemoveSubscriber(identity)
			if len(leaf.Subscribers()) == 0 && leaf.GetRetain() == nil {
				t.foliage.Delete(topic)
			}
			exist = true
		}
	}
	return
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

func (t *Trie) AddRetain(topic string, packet interface{}) (exist bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	return t.root.AddRetain(topic, keys, index, packet)
}

func (t *Trie) RemoveRetain(topic string) (exist bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	for _, branch := range t.root.GetBranches() {
		if branch.RemoveRetain(topic, keys, index) {
			return true
		}
	}
	return false
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
