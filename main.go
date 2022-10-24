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

	go func() {
		if err = c.Tr.Start(); err != nil {
			c.Log.Fatal(err)
		}
	}()

	go prb.Listen()
	cli.Init(c, prb)
}
