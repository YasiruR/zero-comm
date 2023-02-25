package main

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/core/connection"
	"github.com/YasiruR/didcomm-prober/core/did"
	"github.com/YasiruR/didcomm-prober/core/invitation"
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/domain/models"
	"github.com/YasiruR/didcomm-prober/log"
	"github.com/YasiruR/didcomm-prober/prober"
	"github.com/YasiruR/didcomm-prober/pubsub"
	reqRepZmq "github.com/YasiruR/didcomm-prober/reqrep/zmq"
	zmq "github.com/pebbe/zmq4"
	log2 "log"
	"net"
	"os"
	"strconv"
)

func setConfigs(args *container.Args) *container.Config {
	//hostname := `tcp://127.0.0.1:`
	hn, err := os.Hostname()
	if err != nil {
		log2.Fatal(fmt.Sprintf(`fetching hostname failed - %v`, err))
	}

	ips, err := net.LookupIP(hn)
	if err != nil {
		log2.Fatal(fmt.Sprintf(`fetching ip address failed - %v`, err))
	}

	if len(ips) == 0 {
		log2.Fatal(fmt.Sprintf(`could not find an ip address within the kernel`))
	}
	ip := `tcp://` + ips[0].String() + `:`

	return &container.Config{
		Args:        args,
		Hostname:    ip,
		InvEndpoint: ip + strconv.Itoa(args.Port),
		PubEndpoint: ip + strconv.Itoa(args.PubPort),
		LogLevel:    "DEBUG",
	}
}

func initContainer(cfg *container.Config) *container.Container {
	logger := log.NewLogger(cfg.Args.Verbose, 3)
	packer := crypto.NewPacker(logger)
	km := crypto.NewKeyManager()
	ctx, err := zmq.NewContext()
	if err != nil {
		logger.Fatal(fmt.Sprintf(`zmq context initialization failed - %v`, err))
	}

	c := &container.Container{
		Cfg:          cfg,
		KeyManager:   km,
		Packer:       packer,
		DidAgent:     did.NewHandler(),
		Connector:    connection.NewConnector(),
		OOB:          invitation.NewOOBService(cfg),
		Client:       reqRepZmq.NewClient(ctx),
		Log:          logger,
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

	c.PubSub, err = pubsub.NewAgent(ctx, c)
	if err != nil {
		logger.Fatal(`initializing pubsub group agent failed`, err)
	}

	c.Log.Info(fmt.Sprintf(`didcomm agent initialized with messaging port (%d) and publishing port (%d)`, c.Cfg.Port, c.Cfg.PubPort))
	return c
}
