package main

import (
	"bufio"
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"github.com/tryfix/log"
	"os"
	"strings"
)

const (
	sktPub = `pub`
)

func main() {
	output(`>_< Intruder logging in...`)
	//sktTyp := input(`Socket`)
	addr := input(`Address`)
	topic := input(`Topic`)
	subscribe(addr, topic)
}

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
