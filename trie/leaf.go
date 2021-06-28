package trie

type Leaf struct {
	subscribers map[string]interface{}
	retain      interface{}
	parent      *Branch
}

func NewLeaf(parent *Branch) *Leaf {
	leaf := new(Leaf)
	leaf.retain = nil
	leaf.parent = parent
	leaf.subscribers = make(map[string]interface{})
	return leaf
}

func (l *Leaf) GetRetain() interface{} {
	return l.retain
}

func (l *Leaf) SetRetain(retain interface{}) {
	l.retain = retain
}

func (l *Leaf) RemoveRetain() {
	l.retain = nil
}

func (l *Leaf) AddSubscriber(identity string, subscriber interface{}) {
	l.subscribers[identity] = subscriber
}

func (l *Leaf) RemoveSubscriber(identity string) {
	delete(l.subscribers, identity)
}

func (l *Leaf) Subscribers() map[string]interface{} {
	return l.subscribers
}
