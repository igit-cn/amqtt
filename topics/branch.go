package topics

import (
	"fmt"
	"strings"

	"github.com/werbenhu/amq/clients"
)

type Branch struct {
	Trie     *Trie
	Name     string
	Parent   *Branch
	Leaves   map[string]*Leaf
	Branches map[string]*Branch
}

func NewBranch(trie *Trie) *Branch {
	node := new(Branch)
	node.Leaves = make(map[string]*Leaf)
	node.Branches = make(map[string]*Branch)
	node.Parent = nil
	node.Trie = trie
	return node
}

func (n *Branch) GetLeaves() []*Leaf {
	leaves := make([]*Leaf, 0)
	for _, v := range n.Leaves {
		leaves = append(leaves, v)
	}
	return leaves
}

func (n *Branch) Print() {
	fmt.Printf("name:%s\n", n.Name)
	for _, branch := range n.Branches {
		branch.Print()
	}
}

func (b *Branch) AddBranch(client *clients.Client, topic string, keys []string, index int) {
	key := strings.TrimSpace(keys[index])
	fmt.Printf("AddBranch KEY:%s\n", key)
	branch, ok := b.Branches[key]
	if !ok {
		branch = NewBranch(b.Trie)
		branch.Parent = b
		branch.Name = key
		b.Branches[branch.Name] = branch
	}

	if branch.Name == "#" {
		branch.AddLeaf(client, topic)
	} else if index+1 < len(keys) {
		branch.AddBranch(client, topic, keys, index+1)
	} else {
		branch.AddLeaf(client, topic)
	}
}

func (b *Branch) AddLeaf(client *clients.Client, topic string) *Leaf {
	fmt.Printf("AddLeaf topic:%s b:%+v\n", topic, b)
	leaf, ok := b.Leaves[topic]
	if !ok {
		leaf = NewLeaf(b)
		b.Leaves[topic] = leaf
	}
	if client != nil {
		leaf.Clients[topic] = client
	}
	return leaf
}

func (b *Branch) RemoveLeaf(topic string, clientId string) {
	fmt.Printf("RemoveLeaf topic:%s b:%+v\n", topic, b)
	leaf, ok := b.Leaves[topic]
	if ok {
		leaf.RemoveClient(topic, clientId)
		if len(leaf.Clients) == 0 && leaf.Retain == nil {
			delete(b.Leaves, topic)
		}
	}
}

func (b *Branch) CheckClean() {
	if len(b.Leaves) == 0 && len(b.Branches) == 0 {
		if b.Parent != nil {
			delete(b.Parent.Branches, b.Name)
		}
	}
}

func (b *Branch) ScanRemoveLeaf(clientId string, topic string, keys []string, index int) {
	if b.Name == "#" {
		b.RemoveLeaf(topic, clientId)
	} else if index+1 == len(keys) && (b.Name == "+" || b.Name == keys[index]) {
		b.RemoveLeaf(topic, clientId)
	} else if b.Name == "+" || b.Name == keys[index] {
		for _, branch := range b.Branches {
			branch.ScanRemoveLeaf(clientId, topic, keys, index+1)
		}
	}
	b.CheckClean()
}

func (b *Branch) AddRetain(topic string, keys []string, index int, retain *RetainPacket) {
	key := strings.TrimSpace(keys[index])
	fmt.Printf("AddRetain KEY:%s\n", key)
	branch, ok := b.Branches[key]
	if !ok {
		branch = NewBranch(b.Trie)
		branch.Parent = b
		branch.Name = key
		b.Branches[branch.Name] = branch
	}

	if index+1 < len(keys) {
		branch.AddRetain(topic, keys, index+1, retain)
	} else {
		leaf := branch.AddLeaf(nil, topic)
		leaf.SetRetain(retain)
	}
}

func (b *Branch) RemoveRetain(topic string, keys []string, index int) {
	fmt.Printf("RemoveRetain KEY:%s\n", keys[index])
	if index+1 == len(keys) && b.Name == keys[index] {
		for k := range b.Leaves {
			b.Leaves[k].RemoveRetain()
		}
	} else if b.Name == keys[index] {
		for _, branch := range b.Branches {
			branch.RemoveRetain(topic, keys, index+1)
		}
	}
}

func (b *Branch) GetRestRetain() []*RetainPacket {
	retains := make([]*RetainPacket, 0)
	for k := range b.Leaves {
		if b.Leaves[k].Retain != nil {
			retains = append(retains, b.Leaves[k].Retain)
		}
	}
	for _, branch := range b.Branches {
		retains = append(retains, branch.GetRestRetain()...)
	}
	return retains
}

func (b *Branch) SearchRetain(topic string, keys []string, index int) []*RetainPacket {
	retains := make([]*RetainPacket, 0)
	if keys[index] == "#" {
		retains = append(retains, b.GetRestRetain()...)
	} else if index+1 == len(keys) && (keys[index] == "+" || b.Name == keys[index]) {
		for k := range b.Leaves {
			if b.Leaves[k].Retain != nil {
				retains = append(retains, b.Leaves[k].Retain)
			}
		}
	} else if keys[index] == "+" || b.Name == keys[index] {
		for _, branch := range b.Branches {
			retains = append(retains, branch.SearchRetain(topic, keys, index+1)...)
		}
	}
	return retains
}

func (b *Branch) SearchLeaves(topic string, keys []string, index int) []*Leaf {
	fmt.Printf("SearchLeaves key:%s, name:%s\n", keys[index], b.Name)
	leaves := make([]*Leaf, 0)
	if b.Name == "#" {
		return b.GetLeaves()
	} else if index+1 == len(keys) && (b.Name == "+" || b.Name == keys[index]) {
		return b.GetLeaves()
	} else if b.Name == "+" || b.Name == keys[index] {
		for _, branch := range b.Branches {
			leaves = append(leaves, branch.SearchLeaves(topic, keys, index+1)...)
		}
	}
	return leaves
}

func (b *Branch) AddLeaves(topic string, keys []string, index int) []*Leaf {
	leaves := make([]*Leaf, 0)
	if b.Name == "#" {
		return b.GetLeaves()
	} else if index+1 == len(keys) && (b.Name == "+" || b.Name == keys[index]) {
		return b.GetLeaves()
	} else if b.Name == "+" || b.Name == keys[index] {
		for _, branch := range b.Branches {
			leaves = append(leaves, branch.AddLeaves(topic, keys, index+1)...)
		}
	}
	return leaves
}
