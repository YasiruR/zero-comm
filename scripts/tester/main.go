package main

import (
	"fmt"
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
	cfgs := setGroup()
	fmt.Println(`----- START -----`)

	switch typ {
	case testJoin:
		fmt.Printf("[Join latency test for an initial group size of %d with %dms zmq buffer]\n", cfgs.initSize, cfgs.zmqBuf)
		fmt.Println("Test debug logs:")
		avg := joinLatency(cfgs.name, int(cfgs.zmqBuf), true)
		fmt.Printf("Average join-latency (ms): %f\n", avg)
		persist(testJoin, cfgs, []float64{avg})
	default:
		log.Fatalln(`invalid test method`)
	}

	fmt.Println(`----- END -----`)

	// todo throughput
}
