package trie

type Leaf struct {
	Subscribers map[string]interface{}
	Retain  interface{}
	Parent  *Branch
}

func NewLeaf(parent *Branch) *Leaf {
	leaf := new(Leaf)
	leaf.Retain = nil
	leaf.Parent = parent
	leaf.Subscribers = make(map[string]interface{})
	return leaf
}

func (l *Leaf) SetRetain(retain interface{}) {
	l.Retain = retain
}

func (l *Leaf) RemoveRetain() {
	l.Retain = nil
}

func (l *Leaf) RemoveSubscriber(identity string) {
	delete(l.Subscribers, identity)
}
