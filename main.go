package main

import (
	"encoding/base64"
	"github.com/YasiruR/didcomm-prober/cli"
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/prober"
	"github.com/YasiruR/didcomm-prober/transport"
	"github.com/tryfix/log"
)

// did resolver

func main() {
	logger := log.Constructor.Log(log.WithColors(true), log.WithLevel("DEBUG"), log.WithFilePath(true))
	cfg := cli.ParseArgs()

	recChan := make(chan string)
	enc := crypto.NewPacker(logger)
	km := crypto.KeyManager{}
	if err := km.GenerateKeys(); err != nil {
		logger.Fatal(err)
	}

	encodedKey := make([]byte, 64)
	base64.StdEncoding.Encode(encodedKey, km.PublicKey())

	tr := transport.NewHTTP(cfg.Port, enc, &km, recChan, logger)
	go tr.Start()

	prb, err := prober.NewProber(tr, enc, &km, logger)
	if err != nil {
		log.Fatal(err)
	}

	cli.Init(cfg, prb, encodedKey, recChan)
}
