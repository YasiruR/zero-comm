package domain

const (
	ServcDIDExchange = `did-exchange-service`
	ServcMessage     = `message-service`
	ServcGroupJoin   = `group-join-service`
)

const (
	InvitationEndpoint = `` // used in zmq for all endpoints
	ExchangeEndpoint   = `/did-exchange/`
	OOBEndpoint        = `/oob/`
	KillEndpoint       = `/kill`
)

type Role int

const (
	RolePublisher Role = iota
	RoleSubscriber
)
