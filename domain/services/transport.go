package services

import "github.com/YasiruR/didcomm-prober/domain/models"

/* client-server interfaces */

type Transporter interface {
	Client
	Server
}

type Client interface {
	// Send transmits the message but marshalling should be independent of the
	// transport layer to support multiple encoding mechanisms
	Send(typ string, data []byte, endpoint string) (res string, err error) // todo remove res
}

type Server interface {
	// Start should fail for the underlying transport failures
	Start() error
	// AddHandler creates a stream with a notifier for incoming messages.
	// Handlers with synchronous responses can be added by setting async
	// flag to false and handling reply channel in models.Message
	AddHandler(msgType string, notifier chan models.Message, async bool)
	RemoveHandler(msgType string)
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

type PubSubAgent interface {
}
