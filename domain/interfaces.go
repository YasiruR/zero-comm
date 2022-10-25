package domain

/* core services */

type DIDCommService interface {
	Invite() (url string, err error)
	Accept(encodedInv string) error
	SendMessage(text string) error
	ReadMessage(data []byte) error
}

type DIDService interface {
	CreateDIDDoc(endpoint, typ string, encPubKey []byte) DIDDocument
	CreatePeerDID(doc DIDDocument) (did string, err error)
	ValidatePeerDID(did string) error
	CreateConnReq(pthid, did string, encDidDoc AuthCryptMsg) (ConnReq, error)
	ParseConnReq(data []byte) (thId, peerDid string, encDocBytes []byte, err error)
	CreateConnRes(thId, did string, encDidDoc AuthCryptMsg) (ConnRes, error)
	ParseConnRes(data []byte) (thId string, encDocBytes []byte, err error)
}

type OOBService interface {
	CreateInv(did string, didDoc DIDDocument) (url string, err error)
	ParseInv(encInv string) (inv Invitation, endpoint string, pubKey []byte, err error)
}

/* dependencies */

type Transporter interface {
	// Start should fail for the underlying transport failures
	Start()
	// Send transmits the message but marshalling should be independent of the
	// transport layer to support multiple encoding mechanisms
	Send(data []byte, endpoint string) error
	Stop() error
}

type Packer interface {
	Pack(input []byte, recPubKey, sendPubKey, sendPrvKey []byte) (AuthCryptMsg, error)
	Unpack(data, recPubKey, recPrvKey []byte) (output []byte, err error)
}

type Encryptor interface {
	Box(payload, nonce, peerPubKey, mySecKey []byte) (encMsg []byte, err error)
	BoxOpen(cipher, nonce, peerPubKey, mySecKey []byte) (msg []byte, err error)
	SealBox(payload, peerPubKey []byte) (encMsg []byte, err error)
	SealBoxOpen(cipher, peerPubKey, mySecKey []byte) (msg []byte, err error)
	EncryptDetached(msg string, nonce, key []byte) (cipher, mac []byte, err error)
	DecryptDetached(cipher, mac, nonce, key []byte) (msg []byte, err error)
}

type KeyService interface {
	GenerateKeys() error
	PublicKey() []byte
	PrivateKey() []byte
	GenerateInvKeys() error
	InvPublicKey() []byte
	InvPrivateKey() []byte
}
