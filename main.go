package main

import (
	"github.com/YasiruR/didcomm-prober/cli"
)

func main() {
	args := cli.ParseArgs()
	cfg := setConfigs(args)
	c := initContainer(cfg)

	go c.Tr.Start()
	defer shutdown(c)
	cli.Init(c)
}
