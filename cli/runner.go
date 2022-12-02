package cli

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/YasiruR/didcomm-prober/core/discovery"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/domain/services"
	"github.com/tryfix/log"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
)

type runner struct {
	cfg     *domain.Config
	reader  *bufio.Reader
	prober  services.DIDComm
	pub     services.Publisher
	sub     services.Subscriber
	disc    services.Discoverer
	log     log.Logger
	outChan chan string
	disCmds uint64 // flag to identify whether output cursor is on basic commands or not
}

func ParseArgs() *domain.Args {
	n := flag.String(`label`, ``, `agent's name'`)
	p := flag.Int(`port`, 0, `agent's port'`)
	pub := flag.Int(`pub`, 0, `agent's publishing port'`)
	v := flag.Bool(`v`, false, `logging`)
	flag.Parse()

	return &domain.Args{Name: *n, Port: *p, PubPort: *pub, Verbose: *v}
}

func Init(c *domain.Container) {
	fmt.Printf("-> Agent initialized with following attributes: \n\t- Name: %s\n\t- Hostname: %s\n", c.Cfg.Args.Name, c.Cfg.Hostname[:len(c.Cfg.Hostname)-1])
	fmt.Printf("-> Press c and enter for commands\n")

	r := &runner{
		cfg:     c.Cfg,
		reader:  bufio.NewReader(os.Stdin),
		prober:  c.Prober,
		pub:     c.Pub,
		sub:     c.Sub,
		disc:    discovery.NewDiscoverer(c),
		outChan: c.OutChan,
		log:     c.Log,
	}

	go r.listen()
	//r.basicCommands()
	r.enableCommands()
}

/* basic functions */

func (r *runner) basicCommands() {
basicCmds:
	fmt.Printf("\n-> Enter the corresponding number of a command to proceed;\n\t" +
		"[1] Generate invitation\n\t" +
		"[2] Connect via invitation\n\t" +
		"[3] Send a message\n\t" +
		"[4] Register a publisher\n\t" +
		"[5] Set a subscriber\n\t" +
		"[6] Publish a message\n\t" +
		"[7] Unregister\n\t" +
		"[8] Unsubscribe\n\t" +
		"[9] Discover Features\n\t" +
		"[0] Exit\n   Command: ")
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
		r.subscribe()
	case "6":
		r.publishMsg()
	case "7":
		r.unregister()
	case "8":
		r.unsubscribe()
	case "9":
		r.discover()
	case "0":
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

func (r *runner) listen() {
	for {
		text := <-r.outChan
		if r.disCmds == 1 {
			atomic.StoreUint64(&r.disCmds, 0)
			fmt.Println()
		}
		r.output(text)
	}
}

/* command specific functions */

func (r *runner) generateInvitation() {
	inv, err := r.prober.Invite()
	if err != nil {
		r.error(`generating invitation failed`, err)
		return
	}

	r.output(fmt.Sprintf("Invitation URL: %s", inv))
}

func (r *runner) connectWithInv() {
	u, err := url.Parse(strings.TrimSpace(r.input(`Invitation (in URL form)`)))
	if err != nil {
		r.error(`invalid url format, please try again`, err)
		return
	}

	inv, ok := u.Query()[`oob`]
	if !ok {
		r.error(`invitation url must contain 'oob' parameter, please try again`, err)
		return
	}

	if _, err = r.prober.Accept(inv[0]); err != nil {
		r.error(`invitation may be invalid, please try again`, err)
	}
}

func (r *runner) sendMsg() {
	peer := strings.TrimSpace(r.input(`Recipient`))
	msg := strings.TrimSpace(r.input(`Message`))

	if err := r.prober.SendMessage(domain.MsgTypData, peer, msg); err != nil {
		r.error(`sending message failed`, err)
	}
}

func (r *runner) addPublisher() {
	if r.cfg.PubPort == 0 {
		r.error(`unable to initialize a publisher as no specific port is provided`, nil)
		return
	}
	topic := strings.TrimSpace(r.input(`Topic`))
	if err := r.pub.Register(topic); err != nil {
		r.error(`topic may be invalid, please try again`, err)
		return
	}

	r.output(fmt.Sprintf("Publisher registered with topic %s", topic))
}

func (r *runner) subscribe() {
	topic := strings.TrimSpace(r.input(`Topic`))
	strBrokers := r.input(`Brokers (as a comma-separated list)`)
	brokers := strings.Split(strings.TrimSpace(strBrokers), `,`)
	//brokers = []string{`tcp://127.0.0.1:9998`, `tcp://127.0.0.1:9999`}
	r.sub.AddBrokers(topic, brokers)

	if err := r.sub.Subscribe(topic); err != nil {
		r.error(`failed to subscribe, please try again`, err)
	}
}

func (r *runner) publishMsg() {
	topic := strings.TrimSpace(r.input(`Topic`))
	msg := strings.TrimSpace(r.input(`Message`))

	if err := r.pub.Publish(topic, msg); err != nil {
		r.error(`publishing message failed, please try again`, err)
	}
}

func (r *runner) unregister() {
	topic := strings.TrimSpace(r.input(`Topic`))
	if err := r.pub.Unregister(topic); err != nil {
		r.error(`failed to register, please try again`, err)
	}
}

func (r *runner) unsubscribe() {
	topic := strings.TrimSpace(r.input(`Topic`))
	if err := r.sub.Unsubscribe(topic); err != nil {
		r.error(`failed to unsubscribe, please try again`, err)
	}
}

func (r *runner) discover() {
	endpoint := strings.TrimSpace(r.input(`Endpoint`))
	query := strings.TrimSpace(r.input(`Query`))
	comment := strings.TrimSpace(r.input(`Comment`))
	features, err := r.disc.Query(endpoint, query, comment)
	if err != nil {
		r.error(`discovering features failed, please try again`, err)
		return
	}

	var list []string
	for _, f := range features {
		list = append(list, fmt.Sprintf(`feature: "%s", roles: %v`, f.Id, f.Roles))
	}
	r.outputList(`Supported features`, list)
}

/* command-line specific functions */

func (r *runner) input(label string) (input string) {
readInput:
	fmt.Printf("   ? %s: ", label)
	msg, err := r.reader.ReadString('\n')
	if err != nil {
		fmt.Printf("   ! Error: reading %s failed, please try again\n", label)
		goto readInput
	}
	return msg
}

func (r *runner) output(text string) {
	fmt.Printf("-> %s\n", text)
}

func (r *runner) outputList(title string, list []string) {
	var out string
	for i, line := range list {
		out += fmt.Sprintf("    %d. %s\n", i+1, line)
	}
	fmt.Printf(fmt.Sprintf("-> %s:\n%s\n", title, out))
}

func (r *runner) error(cmdOut string, err error) {
	fmt.Printf("   ! Error: %s\n", cmdOut)
	if err != nil {
		r.log.Error(err)
	}
}

func (r *runner) cancelCmd(input string) bool {
	if strings.TrimSpace(input) == `b` || strings.TrimSpace(input) == `B` {
		return true
	}
	return false
}
