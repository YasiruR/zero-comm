package crypto

import (
	chacha "github.com/GoKillers/libsodium-go/crypto/aead/chacha20poly1305ietf"
	"github.com/GoKillers/libsodium-go/cryptobox"
)

const (
	nonceBytes = 24
)

type encryptor struct{}

func (e *encryptor) Box(payload, nonce, peerPubKey, mySecKey []byte) (encMsg []byte, err error) {
	nonce = e.modifyNonce(nonce)
	encMsg, _ = cryptobox.CryptoBoxEasy(payload, nonce, peerPubKey, mySecKey)
	return encMsg, nil
}

func (e *encryptor) BoxOpen(cipher, nonce, peerPubKey, mySecKey []byte) (msg []byte, err error) {
	nonce = e.modifyNonce(nonce)
	msg, _ = cryptobox.CryptoBoxOpenEasy(cipher, nonce, peerPubKey, mySecKey)
	return msg, nil
}

func (e *encryptor) SealBox(payload, peerPubKey []byte) (encMsg []byte, err error) {
	encMsg, _ = cryptobox.CryptoBoxSeal(payload, peerPubKey)
	return encMsg, nil
}

func (e *encryptor) SealBoxOpen(cipher, peerPubKey, mySecKey []byte) (msg []byte, err error) {
	msg, _ = cryptobox.CryptoBoxSealOpen(cipher, peerPubKey, mySecKey)
	return msg, nil
}

func (e *encryptor) EncryptDetached(msg string, nonce, key []byte) (cipher, mac []byte, err error) {
	var convertedIv [chacha.NonceBytes]byte
	copy(convertedIv[:], nonce)

	var convertedCek [chacha.KeyBytes]byte
	copy(convertedCek[:], key)

	cipher, mac = chacha.EncryptDetached([]byte(msg), nil, &convertedIv, &convertedCek)
	return cipher, mac, nil
}

func (e *encryptor) DecryptDetached(cipher, mac, nonce, key []byte) (msg []byte, err error) {
	var convertedIv [chacha.NonceBytes]byte
	copy(convertedIv[:], nonce)

	var convertedCek [chacha.KeyBytes]byte
	copy(convertedCek[:], key)
	msg, err = chacha.DecryptDetached(cipher, mac, nil, &convertedIv, &convertedCek)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// modifyNonce adds noise or trim the byte slice to the cek nonce such that it
// satisfies crypto box open function
func (e *encryptor) modifyNonce(nonce []byte) []byte {
	if len(nonce) < nonceBytes {
		for {
			if len(nonce) == nonceBytes {
				break
			}
			nonce = append(nonce, 48)
		}
	} else if len(nonce) > nonceBytes {
		nonce = nonce[:nonceBytes]
	}

	return nonce
}
