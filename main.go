package main

import (
	"github.com/YasiruR/didcomm-prober/cli"
	"github.com/YasiruR/didcomm-prober/prober"
	"github.com/tryfix/log"
)

func main() {
	name, port, verbose := cli.ParseArgs()
	cfg := setConfigs(name, port, verbose)
	c := initContainer(cfg)

	prb, err := prober.NewProber(c)
	if err != nil {
		log.Fatal(err)
	}

	go c.Tr.Start()
	go prb.Listen()
	defer shutdown(c)
	cli.Init(c, prb)
}
