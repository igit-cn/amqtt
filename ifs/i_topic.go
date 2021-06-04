package ifs

type Topic interface {
	Subscribe(topic string, identity string, subscriber interface{}) error
	Unsubscribe(topic string, identity string) error
	Subscribers(topic string) []interface{}

	AddRetain(topic string, packet interface{}) error
	RemoveRetain(topic string) error
	SearchRetain(topic string)([]interface{}, error)
}
