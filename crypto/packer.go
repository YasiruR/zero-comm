package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	chacha "github.com/GoKillers/libsodium-go/crypto/aead/chacha20poly1305ietf"
	"github.com/GoKillers/libsodium-go/cryptobox"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/btcsuite/btcutil/base58"
	"github.com/tryfix/log"
	"golang.org/x/crypto/nacl/box"
	rand2 "math/rand"
	"strconv"
)

const (
	nonceSize        = 24
	curve2551KeySize = 32
)

type Encryptor struct {
	logger log.Logger
}

func NewEncryptor(logger log.Logger) *Encryptor {
	return &Encryptor{logger: logger}
}

func (e *Encryptor) Pack(msg string, peerPubKey, sendPubKey, sendPrvKey []byte) (domain.AuthCryptMsg, error) {
	// todo check if recipient is set
	// generating and encoding the nonce
	cekIv := []byte(strconv.Itoa(rand2.Int()))
	encodedCekIv := base64.StdEncoding.EncodeToString(cekIv)
	//iv := base64.StdEncoding.EncodeToString(cekIv)

	// generating content encryption key
	cek := make([]byte, 64)
	_, err := rand.Read(cek)
	if err != nil {
		return domain.AuthCryptMsg{}, err
	}

	// todo remove
	for i := 0; i < 5; i++ {
		cekIv = append(cekIv, 48)
	}

	// encrypting cek so it will be decrypted by recipient
	encryptedCek, _ := cryptobox.CryptoBox(cek, cekIv, peerPubKey, sendPrvKey)

	// encrypting sender ver key
	encryptedSendKey, _ := cryptobox.CryptoBoxSeal(sendPubKey, peerPubKey)

	// constructing payload
	payload := domain.Payload{
		// enc: "type",
		Typ: "JWM/1.0",
		Alg: "Authcrypt",
		Recipients: []domain.Recipient{
			{
				EncryptedKey: base64.StdEncoding.EncodeToString(encryptedCek),
				Header: domain.Header{
					Kid:    base58.Encode(peerPubKey),
					Iv:     encodedCekIv,
					Sender: base64.StdEncoding.EncodeToString(encryptedSendKey),
				},
			},
		},
	}

	// base64 encoding of the payload
	data, err := json.Marshal(payload)
	if err != nil {
		return domain.AuthCryptMsg{}, err
	}
	protectedVal := base64.StdEncoding.EncodeToString(data)
	//buf := bytes.Buffer{}
	//if err = gob.NewEncoder(&buf).Encode(payload); err != nil {
	//	return domain.AuthCryptMsg{}, err
	//}
	//protectedVal := buf.Bytes()

	// encrypt with chachapoly1305 detached mode
	iv := []byte(strconv.Itoa(rand2.Int()))
	var convertedIv [chacha.NonceBytes]byte
	copy(convertedIv[:], iv)

	var convertedCek [chacha.KeyBytes]byte
	copy(convertedCek[:], cek)

	cipher, mac := chacha.EncryptDetached([]byte(msg), nil, &convertedIv, &convertedCek)

	// constructing the final message
	authCryptMsg := domain.AuthCryptMsg{
		Protected:  protectedVal,
		Iv:         base64.StdEncoding.EncodeToString(iv),
		Ciphertext: base64.StdEncoding.EncodeToString(cipher),
		Tag:        base64.StdEncoding.EncodeToString(mac),
	}

	return authCryptMsg, nil
}

func (e *Encryptor) Unpack(data, recPubKey, recPrvKey []byte) (text string, err error) {
	// unmarshal into authcrypt message
	var msg domain.AuthCryptMsg
	err = json.Unmarshal(data, &msg)
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}

	// decode protected payload
	var payload domain.Payload
	//buf := bytes.NewBuffer([]byte(msg.Protected))
	//err = gob.NewDecoder(buf).Decode(&payload)
	//if err != nil {
	//	e.logger.Error(err)
	//	return ``, err
	//}
	decodedVal, err := base64.StdEncoding.DecodeString(msg.Protected)
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}

	err = json.Unmarshal(decodedVal, &payload)
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}

	if len(payload.Recipients) == 0 {
		return ``, errors.New("no recipients found")
	}
	rec := payload.Recipients[0]

	// decrypt sender verification key
	decodedSendKey, err := base64.StdEncoding.DecodeString(rec.Header.Sender) // note: array length should be checked
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}
	sendPubKey, _ := cryptobox.CryptoBoxSealOpen(decodedSendKey, recPubKey, recPrvKey)

	// decrypt cek
	decodedCek, err := base64.StdEncoding.DecodeString(rec.EncryptedKey) // note: array length should be checked
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}

	cekIv, err := base64.StdEncoding.DecodeString(rec.Header.Iv)
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}

	for i := 0; i < 5; i++ {
		cekIv = append(cekIv, 48)
	}

	cek, _ := cryptobox.CryptoBoxOpen(decodedCek, cekIv, sendPubKey, recPrvKey)

	// decrypt cipher text
	decodedCipher, err := base64.StdEncoding.DecodeString(msg.Ciphertext)
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}

	mac, err := base64.StdEncoding.DecodeString(msg.Tag)
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}

	iv, err := base64.StdEncoding.DecodeString(msg.Iv)
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}
	var convertedIv [chacha.NonceBytes]byte
	copy(convertedIv[:], iv)

	var convertedCek [chacha.KeyBytes]byte
	copy(convertedCek[:], cek)

	textBytes, err := chacha.DecryptDetached(decodedCipher, mac, nil, &convertedIv, &convertedCek)
	if err != nil {
		e.logger.Error(err)
		return ``, err
	}

	return string(textBytes), nil
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
