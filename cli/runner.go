package cli

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/YasiruR/didcomm-prober/did"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/prober"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
)

type runner struct {
	cfg     domain.Config
	reader  *bufio.Reader
	oob     *did.OOBService
	prober  *prober.Prober
	recChan chan string
	disCmds uint64 // flag to identify whether output cursor is on basic commands or not
}

func ParseArgs() domain.Config {
	// todo verbose and log errors
	name := flag.String(`label`, ``, `agent's name'`)
	port := flag.Int(`port`, 0, `agent's port'`)
	flag.Parse()

	return domain.Config{
		Name:     *name,
		Port:     *port,
		Hostname: "http://localhost:" + strconv.Itoa(*port), // todo change later for remote urls
	}
}

func Init(cfg domain.Config, prb *prober.Prober, recChan chan string) {
	encodedKey := make([]byte, 64)
	base64.StdEncoding.Encode(encodedKey, prb.PublicKey())
	fmt.Printf("-> Agent initialized with following attributes: \n\t- Name: %s\n\t- Hostname: %s\n\t- Public key: %s\n", cfg.Name, cfg.Hostname, string(encodedKey))

	r := runner{cfg: cfg, reader: bufio.NewReader(os.Stdin), prober: prb, recChan: recChan}
	go r.listen()
	r.basicCommands()
}

func (r *runner) basicCommands() {
basicCmds:
	fmt.Printf("\n-> Enter the corresponding number of a command to proceed;\n\t[1] Generate invitation\n\t[2] Set recipient via invitation\n\t[3] Set recipient manually\n\t[4] Send a message\n\t[5] Exit\n   Command: ")
	atomic.AddUint64(&r.disCmds, 1)

	cmd, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading command number failed, please try again")
		goto basicCmds
	}

	switch strings.TrimSpace(cmd) {
	case "1":
		r.generateInvitation()
	case "2":
		r.setRecWithInv()
	case "3":
		//r.setRecManually()
	case "4":
		r.sendMsg()
	case "5":
		log.Fatalln(`program exited`)
	default:
		if r.disCmds > 0 {
			fmt.Println("   Error: invalid command number, please try again")
			goto basicCmds
		}
	}

	atomic.StoreUint64(&r.disCmds, 0)
	r.basicCommands()
}

func (r *runner) generateInvitation() {

	inv, err := r.oob.CreateInvitation(r.prober.PeerDID(), r.prober.DIDDoc())
	if err != nil {
		fmt.Println("-> Error: generating invitation failed")
		return
	}

	fmt.Printf("-> Invitation URL: %s\n", inv)
}

//func (r *runner) setRecManually() {
//	fmt.Printf("-> Enter recipient details:\n")
//readName:
//	fmt.Printf("\tName: ")
//	name, err := r.reader.ReadString('\n')
//	if err != nil {
//		fmt.Println("   Error: reading name failed, please try again")
//		goto readName
//	}
//
//readEndpoint:
//	fmt.Printf("\tHostname: ")
//	endpoint, err := r.reader.ReadString('\n')
//	if err != nil {
//		fmt.Println("   Error: reading endpoint failed, please try again")
//		goto readEndpoint
//	}
//
//readPubKey:
//	fmt.Printf("\tPublic key: ")
//	pubKey, err := r.reader.ReadString('\n')
//	if err != nil {
//		fmt.Println("   Error: reading public key failed, please try again")
//		goto readPubKey
//	}
//
//	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(pubKey))
//	if err != nil {
//		fmt.Println("   Error: decoding public key failed, please try again")
//		goto readPubKey
//	}
//
//	r.prober.SetRecipient(strings.TrimSpace(name), strings.TrimSpace(endpoint), key)
//	fmt.Printf("-> Recipient saved {id: %s, endpoint: %s}\n", name, endpoint)
//}

func (r *runner) setRecWithInv() {
readUrl:
	fmt.Printf("-> Provide invitation URL: ")
	rawUrl, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading url failed, please try again")
		goto readUrl
	}

	u, err := url.Parse(strings.TrimSpace(rawUrl))
	if err != nil {
		fmt.Println("   Error: invalid url format, please try again")
		goto readUrl
	}

	inv, ok := u.Query()[`oob`]
	if !ok {
		fmt.Println("   Error: invitation url must contain 'oob' parameter, please try again")
		goto readUrl
	}

	peerDid, peerEndpoint, peerPubKey, err := r.oob.ParseInvitation(inv[0])
	if err != nil {
		fmt.Println("   Error: invalid invitation, please try again")
		goto readUrl
	}

	r.prober.SetRecipient(peerDid, peerEndpoint, peerPubKey)
	fmt.Printf("-> Recipient saved {id: %s, endpoint: %s}\n", peerDid, peerEndpoint)
}

func (r *runner) sendMsg() {
readMsg:
	fmt.Printf("-> Enter message: ")
	msg, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading endpoint failed, please try again")
		goto readMsg
	}

	if err = r.prober.SendMessage(msg); err != nil {
		fmt.Printf("   Error: sending message failed due to %s", err.Error())
	}
	fmt.Printf("-> Message sent\n")
}

func (r *runner) listen() {
	for {
		text := <-r.recChan
		if r.disCmds == 1 {
			atomic.StoreUint64(&r.disCmds, 0)
			fmt.Println()
		}
		fmt.Printf("-> Message received: %s", text)
	}
}
