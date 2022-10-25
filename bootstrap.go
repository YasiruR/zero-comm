package main

import (
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/did"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/log"
	"github.com/YasiruR/didcomm-prober/transport"
	"strconv"
)

func setConfigs(name string, port int, verbose bool) *domain.Config {
	//hostname := `http://localhost:` + strconv.Itoa(port)
	hostname := `tcp://127.0.0.1:` + strconv.Itoa(port)
	return &domain.Config{
		Name:             name,
		Hostname:         hostname,
		Port:             port,
		InvEndpoint:      hostname + domain.InvitationEndpoint,
		ExchangeEndpoint: hostname + domain.InvitationEndpoint,
		MessageEndpoint:  hostname + domain.InvitationEndpoint,
		Verbose:          verbose,
		LogLevel:         "DEBUG",
	}
}

func initContainer(cfg *domain.Config) *domain.Container {
	logger := log.NewLogger(cfg.Verbose)
	packer := crypto.NewPacker(logger)
	km := crypto.KeyManager{}
	if err := km.GenerateKeys(); err != nil {
		logger.Fatal(err)
	}

	c := &domain.Container{
		Cfg:     cfg,
		KS:      &km,
		Packer:  packer,
		DS:      &did.Handler{},
		OOB:     did.NewOOBService(cfg),
		Log:     logger,
		InChan:  make(chan []byte),
		OutChan: make(chan string),
	}

	//c.Tr = transport.NewHTTP(c)
	zmq, err := transport.NewZmq(c)
	if err != nil {
		logger.Fatal(err)
	}
	c.Tr = zmq

	return c
}

func shutdown(c *domain.Container) {
	c.Tr.Stop()
}
