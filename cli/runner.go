package cli

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/prober"
	log2 "github.com/tryfix/log"
	"log"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
)

type runner struct {
	cfg     *domain.Config
	reader  *bufio.Reader
	prober  *prober.Prober
	log     log2.Logger
	outChan chan string
	disCmds uint64 // flag to identify whether output cursor is on basic commands or not
}

func ParseArgs() (name string, port int, verbose bool) {
	n := flag.String(`label`, ``, `agent's name'`)
	p := flag.Int(`port`, 0, `agent's port'`)
	v := flag.Bool(`v`, false, `logging`)
	flag.Parse()

	return *n, *p, *v
}

func Init(c *domain.Container, prb *prober.Prober) {
	encodedKey := make([]byte, 64)
	base64.StdEncoding.Encode(encodedKey, prb.PublicKey())
	fmt.Printf("-> Agent initialized with following attributes: \n\t- Name: %s\n\t- Hostname: %s\n\t- Public key: %s\n", c.Cfg.Name, c.Cfg.Hostname, string(encodedKey))

	r := runner{cfg: c.Cfg, reader: bufio.NewReader(os.Stdin), prober: prb, log: c.Log, outChan: c.OutChan}
	go r.listen()
	r.basicCommands()
}

func (r *runner) basicCommands() {
basicCmds:
	fmt.Printf("\n-> Enter the corresponding number of a command to proceed;\n\t[1] Generate invitation\n\t[2] Connect via invitation\n\t[3] Send a message\n\t[4] Exit\n   Command: ")
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
		r.connectWithInv()
	case "3":
		r.sendMsg()
	case "4":
		log.Fatalln(`program exited`)
	default:
		if r.disCmds > 0 {
			fmt.Println("   Error: invalid command number, please try again")
			goto basicCmds
		}
	}

	atomic.StoreUint64(&r.disCmds, 0)
	r.enableCommands()
}

func (r *runner) enableCommands() {
	input, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading instruction failed, please try again")
	}

	if strings.TrimSpace(input) == `c` {
		r.basicCommands()
	} else {
		r.enableCommands()
	}
}

func (r *runner) generateInvitation() {
	inv, err := r.prober.Invite()
	if err != nil {
		fmt.Println("-> Error: generating invitation failed")
		return
	}

	fmt.Printf("-> Invitation URL: %s\n", inv)
}

func (r *runner) connectWithInv() {
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

	if err = r.prober.Accept(inv[0]); err != nil {
		fmt.Println("   Error: invitation may be invalid, please try again")
		r.log.Error(err)
		goto readUrl
	}
}

func (r *runner) sendMsg() {
readName:
	fmt.Printf("-> Enter recipient: ")
	peer, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading recipient failed, please try again")
		goto readName
	}

readMsg:
	fmt.Printf("-> Enter message: ")
	msg, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading endpoint failed, please try again")
		goto readMsg
	}

	if err = r.prober.SendMessage(strings.TrimSpace(peer), msg); err != nil {
		fmt.Printf("   Error: sending message failed due to %s", err.Error())
	}
}

func (r *runner) listen() {
	for {
		text := <-r.outChan
		if r.disCmds == 1 {
			atomic.StoreUint64(&r.disCmds, 0)
			fmt.Println()
		}
		fmt.Printf("-> Message received: %s", text)
	}
}
