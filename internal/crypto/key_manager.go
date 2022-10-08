package crypto

import (
	"crypto/rand"
	"crypto/rsa"
)

type KeyManager struct {
	pubKey   *rsa.PublicKey
	prvKey   *rsa.PrivateKey
	peerKeys map[string][]byte
}

func (k *KeyManager) Generate() error {
	pk, err := rsa.GenerateKey(rand.Reader, 256)
	if err != nil {
		return err
	}

	k.pubKey = &pk.PublicKey
	k.prvKey = pk
	return nil
}

func (k *KeyManager) PrivateKey() []byte {
	return k.prvKey.D.Bytes()
}

func (k *KeyManager) PublicKey() []byte {
	return k.pubKey.N.Bytes()
}

func (k *KeyManager) SetPeerKey(label string, peerKey []byte) {
	k.peerKeys[label] = peerKey
}

func (k *KeyManager) PeerKey(label string) []byte {
	return k.peerKeys[label]
}
