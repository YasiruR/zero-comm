package cli

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/tryfix/log"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
)

type runner struct {
	cfg     *domain.Config
	reader  *bufio.Reader
	prober  domain.DIDCommService
	pub     domain.Publisher
	sub     domain.Subscriber
	log     log.Logger
	outChan chan string
	disCmds uint64 // flag to identify whether output cursor is on basic commands or not
}

func ParseArgs() *domain.Args {
	n := flag.String(`label`, ``, `agent's name'`)
	p := flag.Int(`port`, 0, `agent's port'`)
	pub := flag.Int(`pub_port`, 0, `agent's publishing port'`)
	v := flag.Bool(`v`, false, `logging`)
	flag.Parse()

	return &domain.Args{Name: *n, Port: *p, PubPort: *pub, Verbose: *v}
}

func Init(c *domain.Container) {
	fmt.Printf("-> Agent initialized with following attributes: \n\t- Name: %s\n\t- Hostname: %s\n", c.Cfg.Args.Name, c.Cfg.Hostname)
	fmt.Printf("-> Press c and enter for commands\n")
	r := runner{cfg: c.Cfg, reader: bufio.NewReader(os.Stdin), prober: c.Prober, log: c.Log, outChan: c.OutChan}
	go r.listen()
	//r.basicCommands()
	r.enableCommands()
}

func (r *runner) basicCommands() {
basicCmds:
	fmt.Printf("\n-> Enter the corresponding number of a command to proceed;\n\t" +
		"[1] Generate invitation\n\t" +
		"[2] Connect via invitation\n\t" +
		"[3] Send a message\n\t" +
		"[4] Add a publisher\n\t" +
		"[5] Add a subscriber\n\t" +
		"[6] Publish a message\n\t" +
		"[7] Exit\n   Command: ")
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
		r.addPublisher()
	case "5":
		r.addSubscriber()
	case "6":
		r.publishMsg()
	case "7":
		fmt.Println(`program exited`)
		os.Exit(0)
	default:
		if r.disCmds > 0 {
			fmt.Println("   Error: invalid command number, please try again")
			goto basicCmds
		}
	}

	atomic.StoreUint64(&r.disCmds, 0)
	//r.basicCommands()
	r.enableCommands()
}

func (r *runner) enableCommands() {
	input, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading instruction failed, please try again")
	}

	if strings.TrimSpace(input) == `c` || strings.TrimSpace(input) == `C` {
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

	if err = r.prober.SendMessage(domain.MsgTypData, strings.TrimSpace(peer), msg); err != nil {
		fmt.Printf("   Error: sending message failed due to %s", err.Error())
	}
}

func (r *runner) addPublisher() {
readTopic:
	fmt.Printf("-> Topic: ")
	topic, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading topic failed, please try again")
		goto readTopic
	}

	if err = r.pub.Register(topic); err != nil {
		fmt.Println("   Error: topic may be invalid, please try again")
		r.log.Error(err)
		goto readTopic
	}

	fmt.Printf("-> Publisher is registered")
}

func (r *runner) addSubscriber() {
readTopic:
	fmt.Printf("-> Topic: ")
	topic, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading topic failed, please try again")
		goto readTopic
	}

readBrokers:
	fmt.Printf("-> Brokers (as a comma-separated list): ")
	strBrokers, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading brokers failed, please try again")
		goto readBrokers
	}

	brokers := strings.Split(strBrokers, `,`)
	r.sub.AddBrokers(topic, brokers)

	if err = r.sub.Subscribe(topic); err != nil {
		fmt.Println("   Error: topic may be invalid, please try again")
		r.log.Error(err)
		goto readTopic
	}

	fmt.Printf("-> Subscribed to %s", topic)
}

func (r *runner) publishMsg() {
readTopic:
	fmt.Printf("-> Topic: ")
	topic, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading topic failed, please try again")
		goto readTopic
	}

readMsg:
	fmt.Printf("-> Enter message: ")
	msg, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Println("   Error: reading endpoint failed, please try again")
		goto readMsg
	}

	if err = r.pub.Publish(topic, msg); err != nil {
		fmt.Println("   Error: publishing message failed, please try again")
		r.log.Error(err)
		goto readTopic
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
