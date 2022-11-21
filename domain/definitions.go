package domain

// not used for zmq transport
const (
	//InvitationEndpoint = `/invitation/`
	InvitationEndpoint = ``
	ExchangeEndpoint   = `/did-exchange/`
)

const (
	MsgTypConnReq   = `conn-req`
	MsgTypConnRes   = `conn-res`
	MsgTypData      = `data`
	MsgTypSubscribe = `subscribe`
)

const (
	PubTopicSuffix = `_pubs`
)
