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

func (l *Leaf) SetRetain(retain interface{}) bool {
	ret := false
	if l.retain != nil {
		ret = true
	}
	l.retain = retain
	return ret
}

func (l *Leaf) RemoveRetain() bool {
	if l.retain == nil {
		return false
	}
	l.retain = nil
	return true
}

func (l *Leaf) AddSubscriber(identity string, subscriber interface{}) (exist bool) {
	if _, ok := l.subscribers[identity]; ok {
		exist = true
	}
	l.subscribers[identity] = subscriber
	return exist
}

func (l *Leaf) RemoveSubscriber(identity string) {
	delete(l.subscribers, identity)
}

func (l *Leaf) Subscribers() map[string]interface{} {
	return l.subscribers
}
