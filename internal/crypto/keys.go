package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
)

type KeyManager struct {
	pubKey *rsa.PublicKey
	prvKey *rsa.PrivateKey
}

func (k *KeyManager) GenerateKeys() error {
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
	//return k.pubKey.N.Bytes()
	return x509.MarshalPKCS1PublicKey(k.pubKey)
}
