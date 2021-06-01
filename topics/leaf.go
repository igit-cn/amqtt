package topics

import (
	"fmt"

	"github.com/werbenhu/amq/clients"
)

type Leaf struct {
	Trie    *Trie
	Clients map[string]*clients.Client
	Retain  *RetainPacket
	Parent  *Branch
}

func NewLeaf(parent *Branch) *Leaf {
	leaf := new(Leaf)
	leaf.Retain = nil
	leaf.Parent = parent
	if parent != nil {
		leaf.Trie = parent.Trie
	}
	leaf.Clients = make(map[string]*clients.Client)
	return leaf
}

func (l *Leaf) SetRetain(retain *RetainPacket) {
	l.Retain = retain
}

func (l *Leaf) RemoveRetain() {
	fmt.Printf("RemoveRetain\n")
	l.Retain = nil
}

func (l *Leaf) RemoveClient(topic string, clientId string) {
	// if l.Retain != nil {
	// 	delete(l.Retain.clients, clientId)
	// }
	delete(l.Clients, clientId)
}
