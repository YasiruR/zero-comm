package cli

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/prober"
	"log"
	"os"
	"strconv"
	"strings"
)

type runner struct {
	cfg    domain.Config
	reader *bufio.Reader
	prober *prober.Prober
}

func ParseArgs() domain.Config {
	name := flag.String(`label`, ``, `agent's name'`)
	port := flag.Int(`port`, 0, `agent's port'`)
	flag.Parse()

	return domain.Config{
		Name:     *name,
		Port:     *port,
		Endpoint: "http://localhost:" + strconv.Itoa(*port),
	}
}

func Init(cfg domain.Config, prb *prober.Prober) {
	fmt.Printf("# Agent initialized with following attributes: \n\t- Name: %s\n\t- Endpoint: localhost:%d\n\t- Public key: %s\n", cfg.Name, cfg.Port, string(prb.PublicKey()))
	r := runner{cfg: cfg, reader: bufio.NewReader(os.Stdin), prober: prb}
	r.basicCommands()
}

func (r *runner) basicCommands() {
basicCmds:
	fmt.Printf("# Enter the corresponding number of a command to proceed:\n\t[1] Set recipient\n\t[2] Send a message\n\t[3] Exit\nCommand: ")
	cmd, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading command number failed, please try again")
		goto basicCmds
	}

	switch strings.TrimSpace(cmd) {
	case "1":
		r.setRecipient()
	case "2":
		r.sendMsg()
	case "3":
		log.Fatalln(`program exited`)
	default:
		fmt.Println("Error: invalid command number, please try again")
		goto basicCmds
	}

	fmt.Println()
	r.basicCommands()
}

func (r *runner) setRecipient() {
	fmt.Printf("# Enter recipient details:\n")
readName:
	fmt.Printf("\tName: ")
	name, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading name failed, please try again")
		goto readName
	}

readEndpoint:
	fmt.Printf("\tEndpoint: ")
	endpoint, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading endpoint failed, please try again")
		goto readEndpoint
	}

readPubKey:
	fmt.Printf("\tPublic key: ")
	pubKey, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading public key failed, please try again")
		goto readPubKey
	}

	r.prober.SetRecipient(strings.TrimSpace(name), strings.TrimSpace(endpoint), []byte(strings.TrimSpace(pubKey)))
	fmt.Printf("recipient saved\n")
}

func (r *runner) sendMsg() {
readMsg:
	fmt.Printf("# Enter message: ")
	msg, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading endpoint failed, please try again")
		goto readMsg
	}

	if err = r.prober.Send(msg); err != nil {
		fmt.Printf("Error: sending message failed due to %s", err.Error())
	}
}
