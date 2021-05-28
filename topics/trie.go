package topics

import (
	"fmt"
	"github.com/werbenhu/amq/clients"
	"strings"
	"sync"
)

func NewTrie() *Trie {
	trie := new(Trie)
	trie.Root = NewBranch()
	trie.Root.Name = "root"
	return trie
}

type Trie struct {
	Root *Branch
	mu   sync.RWMutex
}

func (t *Trie) AddLeaf(topic string) []*Leaf {
	t.mu.Lock()
	defer t.mu.Lock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	leaves := make([]*Leaf, 0)
	for _, branch := range t.Root.Branches {
		leaves = append(leaves, branch.AddLeaves(topic, keys, index)...)
	}
	fmt.Printf("Find leaves:%+v\n", leaves)
	return leaves
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
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	leaves := make([]*Leaf, 0)
	for _, branch := range t.Root.Branches {
		leaves = append(leaves, branch.SearchLeaves(topic, keys, index)...)
	}
	return leaves
}

func (t *Trie) Parse(client *clients.Client, topic string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	keys := strings.Split(topic, "/")
	index := 0
	if strings.TrimSpace(keys[0]) == "" {
		index = 1
	}
	t.Root.AddBranch(client, topic, keys, index)
	t.Print()
	return nil
}

func (t *Trie) Print() {
	t.Root.Print()
}
