package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/nacl/box"
	"sync"
)

type keys struct {
	pub *[32]byte
	prv *[32]byte
}

type KeyManager struct {
	invPubKey *[32]byte
	invPrvKey *[32]byte
	keyStore  *sync.Map       // key: peer label
	invKeys   map[string]keys // key: topic
}

func NewKeyManager() *KeyManager {
	return &KeyManager{keyStore: &sync.Map{}}
}

func (k *KeyManager) GenerateKeys(peer string) error {
	reader := rand.Reader
	pubKey, prvKey, err := box.GenerateKey(reader)
	if err != nil {
		return err
	}

	k.keyStore.Store(peer, keys{pub: pubKey, prv: prvKey})
	return nil
}

func (k *KeyManager) PrivateKey(peer string) ([]byte, error) {
	val, ok := k.keyStore.Load(peer)
	if !ok {
		return nil, fmt.Errorf(`no private key found for the connection with %s`, peer)
	}
	key := val.(keys)

	tmpPrvKey := *key.prv
	return tmpPrvKey[:], nil
}

func (k *KeyManager) PublicKey(peer string) ([]byte, error) {
	val, ok := k.keyStore.Load(peer)
	if !ok {
		return nil, fmt.Errorf(`no public key found for the connection with %s`, peer)
	}
	key := val.(keys)

	tmpPubKey := *key.pub
	return tmpPubKey[:], nil
}

func (k *KeyManager) Peer(pubKey []byte) (name string, err error) {
	k.keyStore.Range(func(key, val any) bool {
		n := key.(string)
		storePubKey, _ := k.PublicKey(n)
		if string(storePubKey) == string(pubKey) {
			name = n
			return false
		}
		return true
	})

	if name != `` {
		return name, nil
	}

	return ``, fmt.Errorf(`could not find the requested public key (base64-encoded: %s)`, base64.StdEncoding.EncodeToString(pubKey))
}

func (k *KeyManager) GenerateInvKeys() error {
	// uses one key-pair for all invitations but can use separate ones for higher security
	if k.invPrvKey != nil && k.invPubKey != nil {
		return nil
	}

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
