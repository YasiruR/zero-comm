package main

import (
	"bufio"
	"fmt"
	"github.com/YasiruR/didcomm-prober/core/connection"
	"github.com/YasiruR/didcomm-prober/core/did"
	"github.com/YasiruR/didcomm-prober/core/invitation"
	"github.com/YasiruR/didcomm-prober/crypto"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/domain/models"
	log3 "github.com/YasiruR/didcomm-prober/log"
	"github.com/YasiruR/didcomm-prober/prober"
	"github.com/YasiruR/didcomm-prober/pubsub"
	reqRepZmq "github.com/YasiruR/didcomm-prober/reqrep/zmq"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	log2 "log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	sktPub = `pub`
)

// todo fake acceptor/member (with an intruder in the group state)

func main() {
	output(`>_< Intruder logging in...`)
	//sktTyp := input(`Socket`)
	addr := input(`Address`)
	topic := input(`Topic`)
	subscribe(addr, topic)
}

/* fake acceptor */

func accept() {

}

/* fake subscriber */

func subscribe(addr, topic string) {
	ctx, err := zmq.NewContext()
	if err != nil {
		log.Fatal(`context failed`, err)
	}

	skt, err := ctx.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(`socket init failed`, err)
	}

	if err = skt.Connect(addr); err != nil {
		log.Fatal(fmt.Sprintf(`connecting to %s failed`, addr), err)
	}

	if err = skt.SetSubscribe(topic); err != nil {
		log.Fatal(fmt.Sprintf(`subscribing to %s failed`, topic), err)
	}

	defer ctx.Term()
	defer skt.Close()
	log.Info(fmt.Sprintf(`connected to %s and subscribed to %s successfully`, addr, topic))

	for {
		msg, err := skt.Recv(0)
		if err != nil {
			log.Error(`zmq receive error`, err)
		}
		log.Trace(`message received`, msg)
	}
}

func output(msg string) {
	fmt.Printf("%s\n", msg)
}

func input(txt string) string {
readInput:
	fmt.Printf(" ? %s: ", txt)
	msg, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Printf("  :( Error: reading %s failed, please try again\n", txt)
		goto readInput
	}
	return strings.TrimSpace(msg)
}

func init() {
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

	logger := log3.NewLogger(true, 3)
	packer := crypto.NewPacker(logger)
	km := crypto.NewKeyManager()
	ctx, err := zmq.NewContext()
	if err != nil {
		logger.Fatal(fmt.Sprintf(`zmq context initialization failed - %v`, err))
	}

	cfg := container.Config{
		Args: &container.Args{
			Name:    "intruder",
			Port:    9090,
			Verbose: true,
			PubPort: 9091,
			SingleQ: false,
		},
		Hostname:    ip,
		InvEndpoint: ip + strconv.Itoa(9090),
		PubEndpoint: ip + strconv.Itoa(9091),
		LogLevel:    "DEBUG",
	}

	c := &container.Container{
		Cfg:          &cfg,
		KeyManager:   km,
		Packer:       packer,
		DidAgent:     did.NewHandler(),
		Connector:    connection.NewConnector(),
		OOB:          invitation.NewOOBService(&cfg),
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
}
