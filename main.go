package main

import (
	"github.com/YasiruR/didcomm-prober/cli"
	"github.com/YasiruR/didcomm-prober/prober"
	"github.com/YasiruR/didcomm-prober/transport"
	"github.com/tryfix/log"
)

// init prober with args recipient name
// output public key

// set recipient{name, endpoint, public key}

// send message

func main() {
	logger := log.Constructor.Log(log.WithColors(true), log.WithLevel("DEBUG"), log.WithFilePath(true))
	cfg := cli.ParseArgs()

	tr := transport.NewHTTP(cfg.Port, logger)
	go tr.Start()

	prb, pubKey, err := prober.NewProber(tr, logger)
	if err != nil {
		log.Fatal(err)
	}

	cli.Init(cfg, prb, pubKey)
}
