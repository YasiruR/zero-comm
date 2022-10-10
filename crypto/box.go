package crypto

import "golang.org/x/crypto/nacl/box"

const (
	nonceSize        = 24
	curve2551KeySize = 32
)

type Encryptor struct {
}

func (e *Encryptor) Box(payload, nonce, peerPubKey, mySecKey []byte) (encMsg []byte, err error) {
	var (
		tmpPubKey [curve2551KeySize]byte
		tmpSecKey [curve2551KeySize]byte
		tmpNonce  [nonceSize]byte
	)

	copy(tmpPubKey[:], peerPubKey)
	copy(tmpSecKey[:], mySecKey)
	copy(tmpNonce[:], nonce)

	encMsg = box.Seal(nil, payload, &tmpNonce, &tmpPubKey, &tmpSecKey)
	return encMsg, nil
}

func (e *Encryptor) SealBox(payload, peerPubKey []byte) (encMsg []byte, err error) {
	return nil, nil
}
