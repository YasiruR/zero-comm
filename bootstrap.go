package main

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/core/connection"
	"github.com/YasiruR/didcomm-prober/core/did"
	"github.com/YasiruR/didcomm-prober/core/invitation"
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/log"
	"github.com/YasiruR/didcomm-prober/prober"
	"github.com/YasiruR/didcomm-prober/pubsub"
	reqRepZmq "github.com/YasiruR/didcomm-prober/reqrep/zmq"
	zmq "github.com/pebbe/zmq4"
	"strconv"
)

func setConfigs(args *domain.Args) *domain.Config {
	//hostname := `http://localhost:`
	hostname := `tcp://127.0.0.1:`
	return &domain.Config{
		Args:             *args,
		Hostname:         hostname,
		InvEndpoint:      hostname + strconv.Itoa(args.Port) + domain.InvitationEndpoint,
		ExchangeEndpoint: hostname + strconv.Itoa(args.Port) + domain.InvitationEndpoint,
		PubEndpoint:      hostname + strconv.Itoa(args.PubPort),
		LogLevel:         "DEBUG",
	}
}

func initContainer(cfg *domain.Config) *domain.Container {
	logger := log.NewLogger(cfg.Args.Verbose)
	packer := crypto.NewPacker(logger)
	km := crypto.NewKeyManager()
	ctx, err := zmq.NewContext()
	if err != nil {
		logger.Fatal(fmt.Sprintf(`zmq context initialization failed - %v`, err))
	}

	c := &domain.Container{
		Cfg:          cfg,
		KeyManager:   km,
		Packer:       packer,
		DidAgent:     did.NewHandler(),
		Connector:    connection.NewConnector(),
		OOB:          invitation.NewOOBService(cfg),
		Client:       reqRepZmq.NewClient(ctx),
		Log:          logger,
		SubChan:      make(chan models.Message),
		ConnDoneChan: make(chan models.Connection),
		OutChan:      make(chan string),
	}

	s, err := reqRepZmq.NewServer(ctx, c)
	if err != nil {
		logger.Fatal(`initializing zmq server failed`, err)
	}
	c.Server = s

	prb, err := prober.NewProber(c)
	if err != nil {
		logger.Fatal(`initializing prober failed`, err)
	}
	c.Prober = prb

	// should be done after prober since it is a dependency
	initZmqPubSub(ctx, c)

	return c
}

func initZmqPubSub(ctx *zmq.Context, c *domain.Container) {
	if c.Cfg.Args.PubPort != 0 {
		pub, err := pubsub.NewPublisher(ctx, c)
		if err != nil {
			c.Log.Fatal(fmt.Sprintf(`initializing zmq publisher failed - %v`, err))
		}
		c.Pub = pub
	}

	sub, err := pubsub.NewSubscriber(ctx, c)
	if err != nil {
		c.Log.Fatal(fmt.Sprintf(`initializing zmq subscriber failed - %v`, err))
	}
	c.Sub = sub
}

func shutdown(c *domain.Container) {
	c.Server.Stop()
}
