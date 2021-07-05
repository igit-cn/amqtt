package ifs

type TopicStorage interface {
	Subscribe(topic string, identity string, subscriber interface{}) (exist bool)
	Unsubscribe(topic string, identity string) (exist bool)
	Subscribers(topic string) []interface{}

	AddRetain(topic string, packet interface{}) (exist bool)
	RemoveRetain(topic string) (exist bool)
	SearchRetain(topic string) ([]interface{}, error)

	RangeTopics(f func(topic, client interface{}) bool)
}
