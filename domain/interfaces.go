package domain

type Transporter interface {
	Start()
	Send(data []byte, endpoint string) error
	Stop() error
}

type Packer interface {
	Pack(msg string, recPubKey, sendPubKey, sendPrvKey []byte) (AuthCryptMsg, error)
	Unpack(data, recPubKey, recPrvKey []byte) (text string, err error)
}

type Encryptor interface {
	Box(payload, nonce, peerPubKey, mySecKey []byte) (encMsg []byte, err error)
	BoxOpen(cipher, nonce, peerPubKey, mySecKey []byte) (msg []byte, err error)
	SealBox(payload, peerPubKey []byte) (encMsg []byte, err error)
	SealBoxOpen(cipher, peerPubKey, mySecKey []byte) (msg []byte, err error)
	EncryptDetached(msg string, nonce, key []byte) (cipher, mac []byte, err error)
	DecryptDetached(cipher, mac, nonce, key []byte) (msg []byte, err error)
}

type DIDResolver interface {
	Resolve()
}
