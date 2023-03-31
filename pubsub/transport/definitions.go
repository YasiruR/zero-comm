package transport

const (
	topicConnect      = `internal-connect`
	topicSubscribe    = `internal-subscribe`
	topicAuthenticate = `internal-authenticate`
	topicTerm         = `internal-terminate`
)

var internalTopics = []string{topicConnect, topicSubscribe, topicAuthenticate, topicTerm}
