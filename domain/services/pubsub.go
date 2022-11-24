package services

/* message queue functions */

type QueueService interface {
	Publisher
	Subscriber
}

type Publisher interface {
	Register(topic string) error
	Publish(topic, msg string) error
	Unregister(topic string) error
	Close() error
}

type Subscriber interface {
	AddBrokers(topic string, brokers []string)
	Subscribe(topic string) error
	Unsubscribe(topic string) error
	Close() error
}
