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
	if len(args) != 3 {
		log.Fatalln(`incorrect number of arguments`)
	}

	typ, strBuf := args[1], args[2]
	buf, err := strconv.ParseInt(strBuf, 10, 64)
	if err != nil {
		log.Fatalln(`invalid zmq buffer`)
	}

	fmt.Println(`----- START -----`)
	switch typ {
	case testJoin:
		tests.Join(buf)
	default:
		log.Fatalln(`invalid test method`)
	}
	fmt.Println(`----- END -----`)

	// todo throughput
}
