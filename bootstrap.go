package main

import (
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/did"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/log"
	"github.com/YasiruR/didcomm-prober/reqrep"
	"strconv"
)

func setConfigs(args *domain.Args) *domain.Config {
	//hostname := `http://localhost:` + strconv.Itoa(args.Port)
	hostname := `tcp://127.0.0.1:` + strconv.Itoa(args.Port)
	return &domain.Config{
		Args: domain.Args{
			Name:    args.Name,
			Port:    args.Port,
			Verbose: args.Verbose,
		},
		Hostname:         hostname,
		InvEndpoint:      hostname + domain.InvitationEndpoint,
		ExchangeEndpoint: hostname + domain.InvitationEndpoint,
		LogLevel:         "DEBUG",
	}
}

func initContainer(cfg *domain.Config) *domain.Container {
	logger := log.NewLogger(cfg.Args.Verbose)
	packer := crypto.NewPacker(logger)
	km := crypto.NewKeyManager()
	//if err := km.GenerateKeys(); err != nil {
	//	logger.Fatal(err)
	//}

	// todo add pub endpoint and topics

	c := &domain.Container{
		Cfg:          cfg,
		KS:           km,
		Packer:       packer,
		DS:           &did.Handler{},
		OOB:          did.NewOOBService(cfg),
		Log:          logger,
		InChan:       make(chan domain.Message),
		SubChan:      make(chan domain.Message),
		ConnDoneChan: make(chan domain.Connection),
		OutChan:      make(chan string),
	}

	//c.Tr = reqrep.NewHTTP(c)
	zmq, err := reqrep.NewZmq(c)
	if err != nil {
		logger.Fatal(err)
	}
	c.Tr = zmq

	return c
}

func shutdown(c *domain.Container) {
	c.Tr.Stop()
}
