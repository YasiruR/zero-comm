package domain

const (
	MsgTypConnReq   = `conn-req`
	MsgTypConnRes   = `conn-res`
	MsgTypData      = `data`
	MsgTypSubscribe = `subscribe`
	MsgTypQuery     = `query`
)

const (
	PubTopicSuffix = `_pubs`
)

// todo remove
// not used for zmq transport
const (
	//InvitationEndpoint = `/invitation/`
	InvitationEndpoint = ``
	ExchangeEndpoint   = `/did-exchange/`
)
