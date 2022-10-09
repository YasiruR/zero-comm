package main

import (
	"flag"
	"github.com/YasiruR/didcomm-prober/cli"
	"github.com/YasiruR/didcomm-prober/internal"
	"github.com/YasiruR/didcomm-prober/internal/transport"
	"github.com/tryfix/log"
)

// init prober with args recipient name
// output public key

// set recipient{name, endpoint, public key}

// send message

func main() {
	logger := log.Constructor.Log(log.WithColors(true), log.WithLevel("DEBUG"), log.WithFilePath(true))
	name := flag.String(`label`, ``, `agent's name'`)
	endpoint := flag.String(`endpoint`, ``, `agent's address'`)
	flag.Parse()

	tr := transport.NewHTTP(10000, logger)
	p, err := internal.NewProber(*endpoint, tr, logger)
	if err != nil {
		log.Fatal(err)
	}

	cli.Init(*name, *endpoint, string(p.PublicKey()))
}
