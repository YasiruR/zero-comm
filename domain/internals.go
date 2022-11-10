package domain

type ChanMsg struct {
	Type string
	Data []byte
}

type ChanConnDone struct {
	Peer   string
	PubKey []byte
}
