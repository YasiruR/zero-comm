package services

import "github.com/YasiruR/didcomm-prober/domain/messages"

/* core services */

type DIDComm interface {
	Invite() (url string, err error)
	Accept(encodedInv string) (sender string, err error)
	SendMessage(typ, to, text string) error
	ReadMessage(typ string, data []byte) (msg string, err error)
}

type DID interface {
	CreateDIDDoc(endpoint, typ string, pubKey []byte) messages.DIDDocument
	CreatePeerDID(doc messages.DIDDocument) (did string, err error)
	ValidatePeerDID(did string) error
	CreateConnReq(label, pthid, did string, encDidDoc messages.AuthCryptMsg) (messages.ConnReq, error)
	ParseConnReq(data []byte) (label, pthId, peerDid string, encDocBytes []byte, err error)
	CreateConnRes(pthId, did string, encDidDoc messages.AuthCryptMsg) (messages.ConnRes, error)
	ParseConnRes(data []byte) (pthId string, encDocBytes []byte, err error)
}

type OOB interface {
	CreateInv(label, did string, didDoc messages.DIDDocument) (url string, err error)
	ParseInv(encInv string) (inv messages.Invitation, endpoint string, pubKey []byte, err error)
}
