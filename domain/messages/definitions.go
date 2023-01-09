package messages

// todo extend to pub sub messages after defining them
const (
	OOBInvitationV1      = `https://didcomm.org/out-of-band/1.0/invitation`
	DIDExchangeReqV1     = `https://didcomm.org/didexchange/1.0/request`
	DIDExchangeResV1     = `https://didcomm.org/didexchange/1.0/response`
	DIDExchangeCompV1    = `https://didcomm.org/didexchange/1.0/complete`
	DiscoverFeatQuery    = `https://didcomm.org/discover-features/1.0/query`
	DiscoverFeatDisclose = `https://didcomm.org/discover-features/1.0/disclose`
	SubscribeV1          = `https://didcomm.org/pub-sub/1.0/subscribe`
	JoinRequestV1        = `https://didcomm.org/pub-sub/1.0/join-request`
	JoinResponseV1       = `https://didcomm.org/pub-sub/1.0/join-response`
	MemberStatusV1       = `https://didcomm.org/pub-sub/1.0/status`
)
