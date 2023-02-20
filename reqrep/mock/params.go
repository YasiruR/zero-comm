package mock

const (
	InvEndpoint     = `/inv`
	ConnectEndpoint = `/oob`
	CreateEndpoint  = `/create`
	JoinEndpoint    = `/join`
	KillEndpoint    = `/kill`
)

type reqCreate struct {
	Topic     string `json:"topic"`
	Publisher bool   `json:"publisher"`
}

type reqJoin struct {
	Topic     string `json:"topic"`
	Acceptor  string `json:"acceptor"`
	Publisher bool   `json:"publisher"`
}
