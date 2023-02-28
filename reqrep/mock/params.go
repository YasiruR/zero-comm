package mock

import "github.com/YasiruR/didcomm-prober/domain/models"

const (
	InvEndpoint     = `/inv`
	ConnectEndpoint = `/oob`
	CreateEndpoint  = `/create`
	JoinEndpoint    = `/join`
	KillEndpoint    = `/kill`
)

type reqCreate struct {
	Topic     string             `json:"topic"`
	Publisher bool               `json:"publisher"`
	Params    models.GroupParams `json:"params"`
}

type reqJoin struct {
	Topic     string `json:"topic"`
	Acceptor  string `json:"acceptor"`
	Publisher bool   `json:"publisher"`
}
