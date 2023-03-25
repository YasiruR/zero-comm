package transport

const (
	typConnect      = `connect`
	typSubscribe    = `subscribe`
	typAuthenticate = `authenticate`
)

var internalTopics = []string{typConnect, typSubscribe, typAuthenticate}
