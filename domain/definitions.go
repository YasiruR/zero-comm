package domain

const (
	MsgTypConnReq     = `conn-req`
	MsgTypConnRes     = `conn-res`
	MsgTypData        = `data`
	MsgTypSubscribe   = `subscribe`
	MsgTypQuery       = `query`
	MsgTypGroupJoin   = `group-join`
	MsgTypGroupStatus = `group-status`
)

const (
	ServcDIDExchange = `did-exchange-service`
	ServcMessage     = `message-service`
	ServcGroupJoin   = `group-join-service`
	ServcPubSub      = `group-message-service`
)

const (
	InvitationEndpoint = ``
	ExchangeEndpoint   = `/did-exchange/`
)
