package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/btcsuite/btcutil/base58"
	"github.com/tryfix/log"
	rand2 "math/rand"
	"strconv"
)

type Packer struct {
	enc domain.Encryptor
	log log.Logger
}

func NewPacker(logger log.Logger) *Packer {
	return &Packer{enc: &encryptor{}, log: logger}
}

func (p *Packer) Pack(input []byte, recPubKey, sendPubKey, sendPrvKey []byte) (domain.AuthCryptMsg, error) {
	// generating and encoding the nonce
	cekIv := []byte(strconv.Itoa(rand2.Int()))
	encodedCekIv := base64.StdEncoding.EncodeToString(cekIv)

	// generating content encryption key
	cek := make([]byte, 64)
	_, err := rand.Read(cek)
	if err != nil {
		return domain.AuthCryptMsg{}, err
	}

	// encrypting cek so it will be decrypted by recipient
	encryptedCek, err := p.enc.Box(cek, cekIv, recPubKey, sendPrvKey)
	if err != nil {
		return domain.AuthCryptMsg{}, err
	}

	// encrypting sender ver key
	encryptedSendKey, err := p.enc.SealBox(sendPubKey, recPubKey)
	if err != nil {
		return domain.AuthCryptMsg{}, err
	}

	// constructing payload
	payload := domain.Payload{
		Enc: "xchacha20poly1305_ietf",
		Typ: "JWM/1.0",
		Alg: "Authcrypt",
		Recipients: []domain.Recipient{
			{
				EncryptedKey: base64.StdEncoding.EncodeToString(encryptedCek),
				Header: domain.Header{
					Kid:    base58.Encode(recPubKey),
					Iv:     encodedCekIv,
					Sender: base64.StdEncoding.EncodeToString(encryptedSendKey),
				},
			},
		},
	}

	// base64 encoding of the payload
	encPayload, err := json.Marshal(payload)
	if err != nil {
		return domain.AuthCryptMsg{}, err
	}
	protectedVal := base64.StdEncoding.EncodeToString(encPayload)

	// encrypt with chachapoly1305 detached mode
	iv := []byte(strconv.Itoa(rand2.Int()))
	cipher, mac, err := p.enc.EncryptDetached(string(input), string(encPayload), iv, cek)
	if err != nil {
		return domain.AuthCryptMsg{}, err
	}

	// constructing the final message
	authCryptMsg := domain.AuthCryptMsg{
		Protected:  protectedVal,
		Iv:         base64.StdEncoding.EncodeToString(iv),
		Ciphertext: base64.StdEncoding.EncodeToString(cipher),
		Tag:        base64.StdEncoding.EncodeToString(mac),
	}

	return authCryptMsg, nil
}

func (p *Packer) Unpack(data, recPubKey, recPrvKey []byte) (output []byte, err error) {
	// unmarshal into authcrypt message
	var msg domain.AuthCryptMsg
	err = json.Unmarshal(data, &msg)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	// decode protected payload
	var payload domain.Payload
	decodedVal, err := base64.StdEncoding.DecodeString(msg.Protected)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	err = json.Unmarshal(decodedVal, &payload)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	if len(payload.Recipients) == 0 {
		return nil, errors.New("no recipients found")
	}
	rec := payload.Recipients[0]

	// decrypt sender verification key
	decodedSendKey, err := base64.StdEncoding.DecodeString(rec.Header.Sender) // note: array length should be checked
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	sendPubKey, err := p.enc.SealBoxOpen(decodedSendKey, recPubKey, recPrvKey)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	// decrypt cek
	decodedCek, err := base64.StdEncoding.DecodeString(rec.EncryptedKey) // note: array length should be checked
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	cekIv, err := base64.StdEncoding.DecodeString(rec.Header.Iv)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	cek, err := p.enc.BoxOpen(decodedCek, cekIv, sendPubKey, recPrvKey)
	if err != nil {
		return nil, err
	}

	// decrypt cipher text
	decodedCipher, err := base64.StdEncoding.DecodeString(msg.Ciphertext)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	mac, err := base64.StdEncoding.DecodeString(msg.Tag)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	iv, err := base64.StdEncoding.DecodeString(msg.Iv)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	output, err = p.enc.DecryptDetached(decodedCipher, mac, decodedVal, iv, cek)
	if err != nil {
		p.log.Error(err)
		return nil, err
	}

	return output, nil
}
