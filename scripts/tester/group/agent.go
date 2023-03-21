package group

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

func InitAgent(name string, port, pubPort int) *container.Container {
	return initContainer(setConfigs(&container.Args{
		Name:    name,
		Port:    port,
		Verbose: false,
		PubPort: pubPort,
	}))
}

func setConfigs(args *container.Args) *container.Config {
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
	logger := log.NewLogger(cfg.Args.Verbose, 2)
	packer := crypto.NewPacker(logger)
	km := crypto.NewKeyManager()
	ctx, err := zmq.NewContext()
	if err != nil {
		logger.Fatal(`test-agent`, fmt.Sprintf(`zmq context initialization failed - %v`, err))
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
		logger.Fatal(`test-agent`, `initializing zmq server failed`, err)
	}
	c.Server = s

	prb, err := prober.NewProber(c)
	if err != nil {
		logger.Fatal(`test-agent`, `initializing prober failed`, err)
	}
	c.Prober = prb

	c.PubSub, err = pubsub.NewAgent(ctx, c)
	if err != nil {
		logger.Fatal(`test-agent`, `initializing pubsub group agent failed`, err)
	}

	return c
}

func Listen(c *container.Container) {
	for {
		_ = <-c.OutChan
		//fmt.Printf("-> %s\n", text)
	}
}
