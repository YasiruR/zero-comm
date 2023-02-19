package main

import (
	"log"
	"os"
	"strconv"
)

func main() {
	args := os.Args
	if len(args) != 5 {
		log.Fatalln(`incorrect number of arguments`)
	}

	typ, strSize, strBuf, mode := args[1], args[2], args[3], args[4]
	buf, err := strconv.ParseInt(strBuf, 10, 64)
	if err != nil {
		log.Fatalln(`invalid buffer: `, err)
	}

	size, err := strconv.ParseInt(strSize, 10, 64)
	if err != nil {
		log.Fatalln(`invalid size: `, err)
	}

	var singleQ bool
	switch mode {
	case `s`:
		singleQ = true
	case `m`:
		singleQ = false
	default:
		log.Fatalln(`incorrect mode`)
	}

	initGroup(int(size))

	switch typ {
	case `join`:
		joinLatency(int(buf), singleQ, true)
	}

	// todo throughput
}
