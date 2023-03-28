package mock

import "github.com/YasiruR/didcomm-prober/domain/models"

const (
	PingEndpoint      = `/ping`
	InvEndpoint       = `/inv`
	ConnectEndpoint   = `/oob`
	CreateEndpoint    = `/create`
	JoinEndpoint      = `/join`
	KillEndpoint      = `/kill`
	GrpMsgAckEndpoint = `/msg-ack`
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
