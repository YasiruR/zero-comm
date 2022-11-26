package services

import "github.com/YasiruR/didcomm-prober/domain/models"

/* client-server interfaces */

type Transporter interface {
	// Start should fail for the underlying transport failures
	Start()
	Send(typ string, data []byte, endpoint string) (msg []string, err error)
	Stop() error
}

type Client interface {
	// Send transmits the message but marshalling should be independent of the
	// transport layer to support multiple encoding mechanisms
	Send(typ string, data []byte, endpoint string) (msg []string, err error)
}

type Server interface {
	Start() error
	AddHandler(name, endpoint string, notifier chan models.Message)
	Stop() error
}

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
