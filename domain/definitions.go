package domain

const (
	ServcDIDExchange = `did-exchange-service`
	ServcMessage     = `message-service`
	ServcGroupJoin   = `group-join-service`
)

const (
	TopicPrefix  = `urn:didcomm-queue:`
	HelloPrefix  = `hello_`
	MsgTerminate = `terminate-zcomm`
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

/* Roles of group members*/

type Role int

const (
	RolePublisher Role = iota
	RoleSubscriber
	RoleNull
)

/* Modes of the solution based on data queues */

type GroupMode string

const (
	SingleQueueMode   GroupMode = `single-queue`
	MultipleQueueMode GroupMode = `multiple-queue`
)

func (g GroupMode) Valid() bool {
	switch g {
	case SingleQueueMode:
		return true
	case MultipleQueueMode:
		return true
	}
	return false
}

// Retry parameters
const (
	RetryCount        = 10
	RetryIntervalMs   = 50
	InternalTimeoutMs = 1000
)
