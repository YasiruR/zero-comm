package domain

const (
	ServcDIDExchange = `did-exchange-service`
	ServcMessage     = `message-service`
	ServcGroupJoin   = `group-join-service`
)

const (
	InvitationEndpoint = `` // used in zmq for all endpoints
	ExchangeEndpoint   = `/did-exchange`
)

type Role int

const (
	RolePublisher Role = iota
	RoleSubscriber
)
