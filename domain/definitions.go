package domain

const (
	ServcDIDExchange = `did-exchange-service`
	ServcMessage     = `message-service`
	ServcGroupJoin   = `group-join-service`
)

const (
	TopicPrefix = `urn:didcomm-queue:`
)

type ConsistencyLevel string

const (
	NoConsistency    ConsistencyLevel = `none`
	JoinConsistent   ConsistencyLevel = `join`
	StrictConsistent ConsistencyLevel = `all`
)

func (c ConsistencyLevel) Valid() bool {
	switch c {
	case NoConsistency:
		return true
	case JoinConsistent:
		return true
	case StrictConsistent:
		return true
	}

	return false
}

type Role int

const (
	RolePublisher Role = iota
	RoleSubscriber
)
