package main

import (
	"github.com/YasiruR/didcomm-prober/cli"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"os"
	"os/signal"
	"syscall"
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

	go func(c *container.Container) {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL)
		<-sig
		c.Stop()
	}(c)

	if c.Cfg.Mocker {
		mock.Start(c)
	}

	cli.Init(c)
}
