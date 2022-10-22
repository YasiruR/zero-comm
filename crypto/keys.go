package crypto

import (
	"crypto/rand"
	"golang.org/x/crypto/nacl/box"
)

type KeyManager struct {
	pubKey    *[32]byte
	prvKey    *[32]byte
	invPubKey *[32]byte
	invPrvKey *[32]byte
}

func (k *KeyManager) GenerateKeys() error {
	reader := rand.Reader
	pubKey, prvKey, err := box.GenerateKey(reader)
	if err != nil {
		return err
	}

	k.prvKey = prvKey
	k.pubKey = pubKey

	return nil
}

func (k *KeyManager) PrivateKey() []byte {
	tmpPrvKey := *k.prvKey
	return tmpPrvKey[:]
}

func (k *KeyManager) PublicKey() []byte {
	tmpPubKey := *k.pubKey
	return tmpPubKey[:]
}

func (k *KeyManager) GenerateInvKeys() error {
	reader := rand.Reader
	pubKey, prvKey, err := box.GenerateKey(reader)
	if err != nil {
		return err
	}

	k.invPrvKey = prvKey
	k.invPubKey = pubKey

	return nil
}

func (k *KeyManager) InvPrivateKey() []byte {
	tmpPrvKey := *k.invPrvKey
	return tmpPrvKey[:]
}

func (k *KeyManager) InvPublicKey() []byte {
	tmpPubKey := *k.invPubKey
	return tmpPubKey[:]
}
