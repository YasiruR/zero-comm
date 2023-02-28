package main

import (
	"log"
	"os"
	"strconv"
)

func main() {
	args := os.Args
	if len(args) != 4 {
		log.Fatalln(`incorrect number of arguments`)
	}

	typ, strSize, strBuf := args[1], args[2], args[3]
	buf, err := strconv.ParseInt(strBuf, 10, 64)
	if err != nil {
		log.Fatalln(`invalid buffer: `, err)
	}

	size, err := strconv.ParseInt(strSize, 10, 64)
	if err != nil {
		log.Fatalln(`invalid size: `, err)
	}

	initGroup(int(size))

	switch typ {
	case `join`:
		joinLatency(int(buf), true)
	}

	// todo throughput
}
