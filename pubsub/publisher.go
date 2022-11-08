package pubsub

type Publisher struct {
}

func NewPublisher() {
	// init pub struct
}

func Register(topic string) {
	// create a pub socket for topic_pubs
	// generate one invitation for the topic
	// publish status with inv to topic_pubs

	// prober receives multiple conn requests
	// for each request, establish didcomm conn
}

func (p *Publisher) Publish() {

}

func (p *Publisher) Close() {

}
