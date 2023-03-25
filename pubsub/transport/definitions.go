package transport

const (
	topicConnect      = `internal-connect`
	topicSubscribe    = `internal-subscribe`
	topicAuthenticate = `internal-authenticate`
)

var internalTopics = []string{topicConnect, topicSubscribe, topicAuthenticate}
