package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/keyset"
	"github.com/tryfix/log"
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
	b := x509.MarshalPKCS1PublicKey(k.pubKey)
	pubBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: b,
	})

	return pubBytes
}

func Generate(logger log.Logger) {
	kh, err := keyset.NewHandle(aead.AES128GCMKeyTemplate())
	if err != nil {
		logger.Fatal(err)
	}

	h, err := kh.Public()
	if err != nil {
		logger.Fatal(err)
	}

	s := kh.String()
	fmt.Println("STRING: ", s)
	fmt.Println("LEN: ", len(s))

	fmt.Println("H STRING: ", h.String())
	fmt.Println("H LEN: ", len(h.String()))
}
