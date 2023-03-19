package main

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/scripts/tester/tests"
	"log"
	"os"
	"strconv"
)

const (
	testJoin = `join`
)

func main() {
	args := os.Args
	if len(args) != 6 {
		log.Fatalln(`incorrect number of arguments`)
	}

	typ, strZmqBuf, strTestBuf, usr, keyPath := args[1], args[2], args[3], args[4], args[5]
	zmqBuf, err := strconv.ParseInt(strZmqBuf, 10, 64)
	if err != nil {
		log.Fatalln(`invalid zmq buffer`)
	}

	testBuf, err := strconv.ParseInt(strTestBuf, 10, 64)
	if err != nil {
		log.Fatalln(`invalid zmq buffer`)
	}

	fmt.Println(`----- START -----`)
	switch typ {
	case testJoin:
		tests.Join(zmqBuf, testBuf, usr, keyPath)
	default:
		log.Fatalln(`invalid test method`)
	}
	fmt.Println(`----- END -----`)

	// todo throughput
}
