package topics

import (
	"github.com/werbenhu/amq/clients"
	"strings"
	"sync"
)

func NewTrie() *Trie {
	trie := new(Trie)
	trie.Root = NewBranch(trie)
	trie.Root.Name = "root"
	return trie
}

type Trie struct {
	mu      sync.RWMutex
	Root    *Branch
	Foliage sync.Map
}

func (t *Trie) AddRetain(topic string, retain *RetainPacket) {
	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	t.Root.AddRetain(topic, keys, index, retain)
}

func (t *Trie) RemoveRetain(topic string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	for _, branch := range t.Root.Branches {
		branch.RemoveRetain(topic, keys, index)
	}
}

func (t *Trie) SearchRetain(topic string) []*RetainPacket {
	retains := make([]*RetainPacket, 0)
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	for _, branch := range t.Root.Branches {
		retains = append(retains, branch.SearchRetain(topic, keys, index)...)
	}
	return retains
}

func (t *Trie) Remove(client *clients.Client, topic string) {

	_, ok := t.Foliage.Load(topic)
	if ok {
		t.Foliage.Delete(topic)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	for _, branch := range t.Root.Branches {
		branch.ScanRemoveLeaf(client.ClientId, topic, keys, index)
	}
}

func (t *Trie) Search(topic string) []*Leaf {
	leaves := make([]*Leaf, 0)

	leaf, ok := t.Foliage.Load(topic)
	if ok {
		leaves = append(leaves, leaf.(*Leaf))
	}

	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	for _, branch := range t.Root.Branches {
		leaves = append(leaves, branch.SearchLeaves(topic, keys, index)...)
	}
	return leaves
}

func (t *Trie) Parse(client *clients.Client, topic string) error {

	if strings.Contains(topic, "#") || strings.Contains(topic, "+") {
		t.mu.Lock()
		defer t.mu.Unlock()
		keys := strings.Split(topic, "/")
		index := 0
		if strings.TrimSpace(keys[0]) == "" {
			index = 1
		}
		t.Root.AddBranch(client, topic, keys, index)
	} else {
		leaf, ok := t.Foliage.Load(topic)
		if !ok {
			leaf = NewLeaf(nil)
			t.Foliage.Store(topic, leaf)
		}
		leaf.(*Leaf).Clients[client.ClientId] = client
	}

	t.Print()
	return nil
}

func (t *Trie) Print() {
	t.Root.Print()
}
