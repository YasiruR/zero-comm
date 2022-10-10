package domain

type Transporter interface {
	Start()
	Send(data []byte, endpoint string) error
	Stop() error
}

type Encryptor interface {
	Box(payload, nonce, peerPubKey, mySecKey []byte) (encMsg []byte, err error)
	SealBox(payload, peerPubKey []byte) (encMsg []byte, err error)
}

type DIDResolver interface {
	Resolve()
}
