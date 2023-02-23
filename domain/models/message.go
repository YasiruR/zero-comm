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
