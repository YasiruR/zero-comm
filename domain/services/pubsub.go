package services

/* message queue functions */

type QueueService interface {
	Publisher
	Subscriber
	Close() error
}

type Publisher interface {
	Register(topic string) error
	Publish(topic, msg string) error
	Unregister(topic string) error
}

type Subscriber interface {
	AddBrokers(topic string, brokers []string)
	Subscribe(topic string) error
	Unsubscribe(topic string) error
}
