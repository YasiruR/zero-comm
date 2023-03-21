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
	if len(args) != 5 {
		log.Fatalln(`incorrect number of arguments`)
	}

	typ, strTestBuf, usr, keyPath := args[1], args[2], args[3], args[4]
	testBuf, err := strconv.ParseInt(strTestBuf, 10, 64)
	if err != nil {
		log.Fatalln(`invalid zmq buffer`)
	}

	fmt.Println(`----- START -----`)
	switch typ {
	case testJoin:
		tests.Join(testBuf, usr, keyPath)
	default:
		log.Fatalln(`invalid test method`)
	}
	fmt.Println(`----- END -----`)

	// todo throughput
}
