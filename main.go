package main

import (
	"github.com/YasiruR/didcomm-prober/cli"
)

func main() {
	args := cli.ParseArgs()
	cfg := setConfigs(args)
	c := initContainer(cfg)

	go func() {
		if err := c.Server.Start(); err != nil {
			c.Log.Fatal(`failed to start the server`, err)
		}
	}()
	defer shutdown(c)
	cli.Init(c)
}
