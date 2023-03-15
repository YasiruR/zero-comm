package main

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/scripts/tester/tests"
	"log"
	"os"
)

const (
	testJoin = `join`
)

func main() {
	args := os.Args
	if len(args) != 2 {
		log.Fatalln(`incorrect number of arguments`)
	}

	typ := args[1]
	//cfg, grp := group.InitGroup()
	fmt.Println(`----- START -----`)

	switch typ {
	case testJoin:
		tests.Join(10)
	default:
		log.Fatalln(`invalid test method`)
	}

	fmt.Println(`----- END -----`)

	// todo throughput
}
