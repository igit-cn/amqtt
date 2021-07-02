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

func (b *Branch) GetLeaves() map[string]*Leaf {
	return b.leaves
}

func (b *Branch) AddBranch(topic string, identity string, subscriber interface{}, keys []string, index int) (exist bool) {
	key := strings.TrimSpace(keys[index])
	branch, ok := b.branches[key]
	if !ok {
		branch = NewBranch(b.trie)
		branch.parent = b
		branch.name = key
		b.branches[branch.name] = branch
	}

	if branch.name == "#" {
		_, exist = branch.AddLeaf(topic, identity, subscriber)
		if branch.parent != nil {
			branch.parent.AddLeaf(topic, identity, subscriber)
		}
	} else if index+1 < len(keys) {
		exist = branch.AddBranch(topic, identity, subscriber, keys, index+1)
	} else {
		_, exist = branch.AddLeaf(topic, identity, subscriber)
	}
	return exist
}

func (b *Branch) AddLeaf(topic string, identity string, subscriber interface{}) (*Leaf, bool) {
	leaf, ok := b.leaves[topic]
	exist := false
	if !ok {
		leaf = NewLeaf(b)
		b.leaves[topic] = leaf
	}
	if subscriber != nil {
		exist = leaf.AddSubscriber(identity, subscriber)
	}
	return leaf, exist
}

func (b *Branch) RemoveLeaf(topic string, identity string) (exist bool) {
	leaf, ok := b.leaves[topic]
	if ok {
		leaf.RemoveSubscriber(identity)
		if len(leaf.Subscribers()) == 0 && leaf.GetRetain() == nil {
			delete(b.leaves, topic)
		}
		return true
	}
	return false
}

func (b *Branch) CheckClean() {
	if len(b.leaves) == 0 && len(b.branches) == 0 {
		if b.parent != nil {
			delete(b.parent.branches, b.name)
		}
	}
}

func (b *Branch) ScanRemoveLeaf(identity string, topic string, keys []string, index int) (exist bool) {
	if b.name == "#" {
		exist = b.RemoveLeaf(topic, identity)
	} else if index+1 == len(keys) && b.name == keys[index] {
		exist = b.RemoveLeaf(topic, identity)
	} else if b.name == keys[index] {
		for _, branch := range b.branches {
			if branch.ScanRemoveLeaf(identity, topic, keys, index+1) {
				exist = true
				break
			}
		}
	}
	if exist {
		b.CheckClean()
	}
	return exist
}

func (b *Branch) AddRetain(topic string, keys []string, index int, retain interface{}) bool {
	key := strings.TrimSpace(keys[index])
	branch, ok := b.branches[key]
	if !ok {
		branch = NewBranch(b.trie)
		branch.parent = b
		branch.name = key
		b.branches[branch.name] = branch
	}

	if index+1 < len(keys) {
		return branch.AddRetain(topic, keys, index+1, retain)
	} else {
		leaf, _ := branch.AddLeaf(topic, "", nil)
		return leaf.SetRetain(retain)
	}
}

func (b *Branch) RemoveRetain(topic string, keys []string, index int) bool {
	if index+1 == len(keys) && b.name == keys[index] {
		for k := range b.leaves {
			if b.leaves[k].RemoveRetain() {
				return true
			}
		}
	} else if b.name == keys[index] {
		for _, branch := range b.branches {
			if branch.RemoveRetain(topic, keys, index+1) {
				return true
			}
		}
	}
	return false
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

func (b *Branch) SearchLeaves(topic string, keys []string, index int) map[string]*Leaf {
	leaves := make(map[string]*Leaf)
	if b.name == "#" {
		return b.GetLeaves()
	} else if index+1 == len(keys) && (b.name == "+" || b.name == keys[index]) {
		return b.GetLeaves()
	} else if b.name == "+" || b.name == keys[index] {
		for _, branch := range b.branches {
			subLeaves := branch.SearchLeaves(topic, keys, index+1)
			for k, v := range subLeaves {
				leaves[k] = v
			}
		}
	}
	return leaves
}
