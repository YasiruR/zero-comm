package main

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/scripts/tester/tests"
	"log"
	"os"
	"strconv"
)

func main() {
	args := os.Args
	if len(args) != 5 && len(args) != 6 {
		log.Fatalf("incorrect number of arguments (provided=%d)\n", len(args))
	}

	typ, strTestBuf, usr, keyPath := args[1], args[2], args[3], args[4]
	testBuf, err := strconv.ParseInt(strTestBuf, 10, 64)
	if err != nil {
		log.Fatalln(`invalid zmq buffer`)
	}

	var manualSize int
	if len(args) == 6 {
		manualSize, err = strconv.Atoi(args[5])
		if err != nil {
			log.Fatalln(`invalid optional parameter`)
		}
	}

	fmt.Println(`----- START -----`)
	switch tests.TestMode(typ) {
	case tests.JoinLatency:
		tests.Join(tests.JoinLatency, testBuf, usr, keyPath, manualSize)
	case tests.JoinThroughput:
		tests.Join(tests.JoinThroughput, testBuf, usr, keyPath, manualSize)
	case tests.PublishLatency:
		tests.Send(testBuf, usr, keyPath, manualSize)
	case tests.MsgSize:
		tests.Size()
	default:
		log.Fatalln(`invalid test method`)
	}
	fmt.Println(`----- END -----`)
}
