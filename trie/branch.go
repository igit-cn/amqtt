package trie

import (
	"strings"
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

func (b *Branch) GetLeaves() []*Leaf {
	leaves := make([]*Leaf, 0)
	for _, v := range b.Leaves {
		leaves = append(leaves, v)
	}
	return leaves
}

func (b *Branch) AddBranch(topic string, identity string, subscriber interface{}, keys []string, index int) {
	key := strings.TrimSpace(keys[index])
	branch, ok := b.Branches[key]
	if !ok {
		branch = NewBranch(b.Trie)
		branch.Parent = b
		branch.Name = key
		b.Branches[branch.Name] = branch
	}

	if branch.Name == "#" {
		branch.AddLeaf(topic, identity, subscriber)
	} else if index+1 < len(keys) {
		branch.AddBranch(topic, identity, subscriber, keys, index+1)
	} else {
		branch.AddLeaf(topic, identity, subscriber)
	}
}

func (b *Branch) AddLeaf(topic string, identity string, subscriber interface{}) *Leaf {
	leaf, ok := b.Leaves[topic]
	if !ok {
		leaf = NewLeaf(b)
		b.Leaves[topic] = leaf
	}
	if subscriber != nil {
		leaf.Subscribers[identity] = subscriber
	}
	return leaf
}

func (b *Branch) RemoveLeaf(topic string, identity string) {
	leaf, ok := b.Leaves[topic]
	if ok {
		leaf.RemoveSubscriber(identity)
		if len(leaf.Subscribers) == 0 && leaf.Retain == nil {
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

func (b *Branch) ScanRemoveLeaf(identity string, topic string, keys []string, index int) {
	if b.Name == "#" {
		b.RemoveLeaf(topic, identity)
	} else if index+1 == len(keys) && (b.Name == "+" || b.Name == keys[index]) {
		b.RemoveLeaf(topic, identity)
	} else if b.Name == "+" || b.Name == keys[index] {
		for _, branch := range b.Branches {
			branch.ScanRemoveLeaf(identity, topic, keys, index+1)
		}
	}
	b.CheckClean()
}

func (b *Branch) AddRetain(topic string, keys []string, index int, retain interface{}) {
	key := strings.TrimSpace(keys[index])
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
		leaf := branch.AddLeaf(topic, "", nil)
		leaf.SetRetain(retain)
	}
}

func (b *Branch) RemoveRetain(topic string, keys []string, index int) {
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

func (b *Branch) GetRestRetain() []interface{} {
	retains := make([]interface{}, 0)
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

func (b *Branch) SearchRetain(topic string, keys []string, index int) []interface{} {
	retains := make([]interface{}, 0)
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
