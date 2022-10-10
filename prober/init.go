package prober

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	chacha "github.com/GoKillers/libsodium-go/crypto/aead/chacha20poly1305ietf"
	"github.com/GoKillers/libsodium-go/cryptobox"
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/btcsuite/btcutil/base58"
	"github.com/tryfix/log"
	rand2 "math/rand"
	"strconv"
)

type recipient struct {
	name      string
	endpoint  string
	publicKey []byte
}

type Prober struct {
	rec         *recipient
	transporter domain.Transporter
	*crypto.KeyManager
	logger log.Logger
}

func NewProber(t domain.Transporter, logger log.Logger) (*Prober, error) {
	km := crypto.KeyManager{}
	if err := km.GenerateKeys(); err != nil {
		logger.Error(err)
		return nil, err
	}

	return &Prober{
		KeyManager:  &km,
		transporter: t,
		logger:      logger,
	}, nil
}

func (p *Prober) SetRecipient(name, endpoint string, key []byte) {
	p.rec = &recipient{name: name, endpoint: endpoint, publicKey: key}
}

func (p *Prober) Send(text string) error {
	msg, err := p.pack(text)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	err = p.transporter.Send(data, p.rec.endpoint)
	if err != nil {
		p.logger.Error(err)
		return err
	}

	return nil
}

func (p *Prober) pack(msg string) (domain.AuthCryptMsg, error) {
	// todo check if recipient is set
	// generating and encoding the nonce
	cekIv := []byte(strconv.Itoa(rand2.Int()))
	iv := base64.StdEncoding.EncodeToString(cekIv)

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

	pubKey := p.rec.publicKey
	fmt.Println("PUB: ", pubKey)
	fmt.Println("PUB LEN: ", len(pubKey))

	// encrypting cek so it will be decrypted by recipient
	encryptedCek, _ := cryptobox.CryptoBox(cek, cekIv, pubKey, p.PrivateKey())

	// encrypting sender ver key
	encryptedSendKey, _ := cryptobox.CryptoBoxSeal(p.PublicKey(), p.rec.publicKey)

	// constructing payload
	payload := domain.Payload{
		// enc: "type",
		Typ: "JWM/1.0",
		Alg: "Authcrypt",
		Recipients: []domain.Recipient{
			{
				EncryptedKey: base64.StdEncoding.EncodeToString(encryptedCek),
				Header: domain.Header{
					Kid:    base58.Encode(p.rec.publicKey),
					Iv:     iv,
					Sender: base64.StdEncoding.EncodeToString(encryptedSendKey),
				},
			},
		},
	}

	// base64 encoding of the payload
	buf := bytes.Buffer{}
	if err = gob.NewEncoder(&buf).Encode(payload); err != nil {
		return domain.AuthCryptMsg{}, err
	}
	protectedVal := buf.String()

	// encrypt with chachapoly1305 detached mode
	var convertedIv [chacha.NonceBytes]byte
	copy(convertedIv[:], []byte(iv))

	var convertedCek [chacha.KeyBytes]byte
	copy(convertedCek[:], cek)

	cipher, mac := chacha.EncryptDetached([]byte(msg), []byte(protectedVal), &convertedIv, &convertedCek)

	// constructing the final message
	authCryptMsg := domain.AuthCryptMsg{
		Protected:  protectedVal,
		Iv:         iv,
		Ciphertext: base64.StdEncoding.EncodeToString(cipher),
		Tag:        base64.StdEncoding.EncodeToString(mac),
	}

	return authCryptMsg, nil
}
