package models

type MsgType int

const (
	TypConnReq MsgType = iota
	TypConnRes
	TypData
	TypSubscribe
	TypQuery
	TypGroupJoin
	TypGroupStatus
	TypGroupMsg
	TypStatusAck
	TypTerminate
)

func (m MsgType) String() string {
	switch m {
	case TypConnReq:
		return `connection-request`
	case TypConnRes:
		return `connection-response`
	case TypData:
		return `data-message`
	case TypSubscribe:
		return `subscribe-request`
	case TypQuery:
		return `query-message`
	case TypGroupJoin:
		return `join-request`
	case TypGroupStatus:
		return `status-message`
	case TypGroupMsg:
		return `group-message`
	case TypStatusAck:
		return `hello-ack`
	case TypTerminate:
		return `internal-terminate-message`
	default:
		return `undefined`
	}
}

type Message struct {
	Type  MsgType
	Data  []byte
	Reply chan []byte
}

type Connection struct {
	Peer     string
	Endpoint string
	PubKey   []byte
}
