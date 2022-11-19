package domain

// may not be useful for zmq transport
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
	pubTopicSuffix = `_pubs`
)
