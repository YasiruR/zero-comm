package transport

const (
	TypStateSkt sktType = iota
	TypMsgSkt
)

const (
	stateEndpoint = `ipc://state.ipc`
	dataEndpoint  = `ipc://data.ipc`
	pubEndpoint   = `ipc://publish.ipc`
)

const (
	typConnect      = `connect`
	typSubscribe    = `subscribe`
	typAuthenticate = `authenticate`
)
