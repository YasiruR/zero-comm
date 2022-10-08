package internal

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	chacha "github.com/GoKillers/libsodium-go/crypto/aead/chacha20poly1305ietf"
	"github.com/GoKillers/libsodium-go/cryptobox"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/internal/crypto"
	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/packager"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport"
	didHttp "github.com/hyperledger/aries-framework-go/pkg/didcomm/transport/http"
	"github.com/hyperledger/aries-framework-go/pkg/framework/context"
	rand2 "math/rand"
	"net/http"
	"strconv"
)

type Prober struct {
	packer   transport.Packager
	inbound  transport.InboundTransport
	outbound transport.OutboundTransport
	*crypto.KeyManager
}

func Init(addr string) (*Prober, error) {
	outClient, err := didHttp.NewOutbound(didHttp.WithOutboundHTTPClient(&http.Client{}))
	if err != nil {
		return nil, err
	}

	inClient, err := didHttp.NewInbound(addr, ``, ``, ``)
	if err != nil {
		return nil, err
	}

	ctxProvider, err := context.New(context.WithOutboundTransports(outClient))
	if err != nil {
		return nil, err
	}

	packer, err := packager.New(ctxProvider)
	if err != nil {
		return nil, err
	}

	return &Prober{
		outbound: outClient,
		inbound:  inClient,
		packer:   packer,
	}, nil
}

func (p *Prober) Pack(msg, recipient string) (domain.AuthCryptMsg, error) {
	peerKey := p.PeerKey(recipient)

	// generating and encoding the nonce
	cekIv := []byte(strconv.Itoa(rand2.Int()))
	iv := base64.StdEncoding.EncodeToString(cekIv)

	// generating content encryption key
	cek := make([]byte, 64)
	_, err := rand.Read(cek)
	if err != nil {
		return domain.AuthCryptMsg{}, err
	}

	// encrypting cek so it will be decrypted by recipient
	encryptedCek, _ := cryptobox.CryptoBox(cek, cekIv, peerKey, p.PrivateKey())

	// encrypting sender ver key
	encryptedSendKey, _ := cryptobox.CryptoBoxSeal(p.PublicKey(), peerKey)

	// constructing payload
	payload := domain.Payload{
		// enc: "type",
		Typ: "JWM/1.0",
		Alg: "Authcrypt",
		Recipients: []domain.Recipient{
			{
				EncryptedKey: base64.StdEncoding.EncodeToString(encryptedCek),
				Header: domain.Header{
					Kid:    base58.Encode(peerKey),
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
