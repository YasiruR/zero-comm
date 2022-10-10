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
	cfg     domain.Config
	reader  *bufio.Reader
	prober  *prober.Prober
	recChan chan string
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

func Init(cfg domain.Config, prb *prober.Prober, pubKey []byte, recChan chan string) {
	fmt.Printf("-> Agent initialized with following attributes: \n\t- Name: %s\n\t- Endpoint: localhost:%d\n\t- Public key: %s\n", cfg.Name, cfg.Port, string(pubKey))
	r := runner{cfg: cfg, reader: bufio.NewReader(os.Stdin), prober: prb, recChan: recChan}
	go r.listen()
	r.basicCommands()
}

func (r *runner) basicCommands() {
basicCmds:
	fmt.Printf("\n\t[1] Set recipient\n\t[2] Send a message\n\t[3] Exit\n-> Enter the corresponding number of a command to proceed: ")
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

	r.basicCommands()
}

func (r *runner) setRecipient() {
	fmt.Printf("-> Enter recipient details:\n")
readName:
	fmt.Printf("\tName: ")
	name, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading name failed, please try again")
		goto readName
	}
	name = `bob`

readEndpoint:
	fmt.Printf("\tEndpoint: ")
	endpoint, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading endpoint failed, please try again")
		goto readEndpoint
	}
	endpoint = `http://localhost:6666`

readPubKey:
	fmt.Printf("\tPublic key: ")
	pubKey, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading public key failed, please try again")
		goto readPubKey
	}

	if err = r.prober.SetRecipient(strings.TrimSpace(name), strings.TrimSpace(endpoint), strings.TrimSpace(pubKey)); err != nil {
		fmt.Println("Error: public key may be invalid, please try again")
		goto readPubKey
	}
	fmt.Printf("-> Recipient saved {name: %s, endpoint: %s}\n", name, endpoint)
}

func (r *runner) sendMsg() {
readMsg:
	fmt.Printf("-> Enter message: ")
	msg, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error: reading endpoint failed, please try again")
		goto readMsg
	}

	if err = r.prober.Send(msg); err != nil {
		fmt.Printf("Error: sending message failed due to %s", err.Error())
	}
	fmt.Printf("-> Message sent\n")
}

func (r *runner) listen() {
	for {
		text := <-r.recChan
		fmt.Printf("\n-> Message received: %s", text)
		r.basicCommands()
	}
}
