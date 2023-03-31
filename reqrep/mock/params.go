package mock

import "github.com/YasiruR/didcomm-prober/domain/models"

const (
	PingEndpoint      = `/ping`
	InvEndpoint       = `/inv`
	ConnectEndpoint   = `/oob`
	CreateEndpoint    = `/create`
	JoinEndpoint      = `/join`
	GrpMsgAckEndpoint = `/msg-ack`
	KillEndpoint      = `/kill`
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

type ReqRegAck struct {
	Peer  string `json:"peer"`
	Msg   string `json:"msg"`
	Count int    `json:"count"`
}

type ReqForceRemv struct {
	Label       string `json:"label"`
	Publisher   bool   `json:"publisher"`
	Topic       string `json:"topic"`
	PubEndpoint string `json:"pubEndpoint"`
}
