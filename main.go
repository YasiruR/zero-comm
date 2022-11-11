package main

import (
	"github.com/YasiruR/didcomm-prober/cli"
	"github.com/YasiruR/didcomm-prober/prober"
	"github.com/tryfix/log"
)

func main() {
	args := cli.ParseArgs()
	cfg := setConfigs(args)
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
