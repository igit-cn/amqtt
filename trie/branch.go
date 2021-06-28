package trie

import (
	"strings"
)

type Branch struct {
	trie     *Trie
	name     string
	parent   *Branch
	leaves   map[string]*Leaf
	branches map[string]*Branch
}

func NewBranch(trie *Trie) *Branch {
	node := new(Branch)
	node.leaves = make(map[string]*Leaf)
	node.branches = make(map[string]*Branch)
	node.parent = nil
	node.trie = trie
	return node
}

func (b *Branch) SetName(name string) {
	b.name = name
}

func (b *Branch) GetBranches() map[string]*Branch {
	return b.branches
}

func (b *Branch) GetLeaves() []*Leaf {
	leaves := make([]*Leaf, 0)
	for _, v := range b.leaves {
		leaves = append(leaves, v)
	}
	return leaves
}

func (b *Branch) AddBranch(topic string, identity string, subscriber interface{}, keys []string, index int) {
	key := strings.TrimSpace(keys[index])
	branch, ok := b.branches[key]
	if !ok {
		branch = NewBranch(b.trie)
		branch.parent = b
		branch.name = key
		b.branches[branch.name] = branch
	}

	if branch.name == "#" {
		branch.AddLeaf(topic, identity, subscriber)
		if branch.parent != nil {
			branch.parent.AddLeaf(topic, identity, subscriber)
		}
	} else if index+1 < len(keys) {
		branch.AddBranch(topic, identity, subscriber, keys, index+1)
	} else {
		branch.AddLeaf(topic, identity, subscriber)
	}
}

func (b *Branch) AddLeaf(topic string, identity string, subscriber interface{}) *Leaf {
	leaf, ok := b.leaves[topic]
	if !ok {
		leaf = NewLeaf(b)
		b.leaves[topic] = leaf
	}
	if subscriber != nil {
		leaf.AddSubscriber(identity, subscriber)
	}
	return leaf
}

func (b *Branch) RemoveLeaf(topic string, identity string) {
	leaf, ok := b.leaves[topic]
	if ok {
		leaf.RemoveSubscriber(identity)
		if len(leaf.Subscribers()) == 0 && leaf.GetRetain() == nil {
			delete(b.leaves, topic)
		}
	}
}

func (b *Branch) CheckClean() {
	if len(b.leaves) == 0 && len(b.branches) == 0 {
		if b.parent != nil {
			delete(b.parent.branches, b.name)
		}
	}
}

func (b *Branch) ScanRemoveLeaf(identity string, topic string, keys []string, index int) {
	if b.name == "#" {
		b.RemoveLeaf(topic, identity)
	} else if index+1 == len(keys) && (b.name == "+" || b.name == keys[index]) {
		b.RemoveLeaf(topic, identity)
	} else if b.name == "+" || b.name == keys[index] {
		for _, branch := range b.branches {
			branch.ScanRemoveLeaf(identity, topic, keys, index+1)
		}
	}
	b.CheckClean()
}

func (b *Branch) AddRetain(topic string, keys []string, index int, retain interface{}) {
	key := strings.TrimSpace(keys[index])
	branch, ok := b.branches[key]
	if !ok {
		branch = NewBranch(b.trie)
		branch.parent = b
		branch.name = key
		b.branches[branch.name] = branch
	}

	if index+1 < len(keys) {
		branch.AddRetain(topic, keys, index+1, retain)
	} else {
		leaf := branch.AddLeaf(topic, "", nil)
		leaf.SetRetain(retain)
	}
}

func (b *Branch) RemoveRetain(topic string, keys []string, index int) {
	if index+1 == len(keys) && b.name == keys[index] {
		for k := range b.leaves {
			b.leaves[k].RemoveRetain()
		}
	} else if b.name == keys[index] {
		for _, branch := range b.branches {
			branch.RemoveRetain(topic, keys, index+1)
		}
	}
}

func (b *Branch) GetRestRetain() []interface{} {
	retains := make([]interface{}, 0)
	for k := range b.leaves {
		if b.leaves[k].GetRetain() != nil {
			retains = append(retains, b.leaves[k].GetRetain())
		}
	}
	for _, branch := range b.branches {
		retains = append(retains, branch.GetRestRetain()...)
	}
	return retains
}

func (b *Branch) SearchRetain(topic string, keys []string, index int) []interface{} {
	retains := make([]interface{}, 0)
	if keys[index] == "#" {
		retains = append(retains, b.GetRestRetain()...)
	} else if index+1 == len(keys) && (keys[index] == "+" || b.name == keys[index]) {
		for k := range b.leaves {
			if b.leaves[k].GetRetain() != nil {
				retains = append(retains, b.leaves[k].GetRetain())
			}
		}
	} else if keys[index] == "+" || b.name == keys[index] {
		for _, branch := range b.branches {
			retains = append(retains, branch.SearchRetain(topic, keys, index+1)...)
		}
	}
	return retains
}

func (b *Branch) SearchLeaves(topic string, keys []string, index int) []*Leaf {
	leaves := make([]*Leaf, 0)
	if b.name == "#" {
		return b.GetLeaves()
	} else if index+1 == len(keys) && (b.name == "+" || b.name == keys[index]) {
		return b.GetLeaves()
	} else if b.name == "+" || b.name == keys[index] {
		for _, branch := range b.branches {
			leaves = append(leaves, branch.SearchLeaves(topic, keys, index+1)...)
		}
	}
	return leaves
}
