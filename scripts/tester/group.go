package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
)

type groupCfg struct {
	name          string
	initSize      int64
	mode          string
	consistntJoin bool
	ordered       bool
	zmqBuf        int64
}

type member struct {
	name         string
	mockEndpoint string
}

var group []member

func setGroup() groupCfg {
	// read group configs
	f1, err := os.Open(`../deployer/group_cfg.csv`)
	if err != nil {
		log.Fatal(`opening group configs failed`, err)
	}
	defer f1.Close()

	var gc groupCfg
	cfgs, err := csv.NewReader(f1).ReadAll()
	for _, row := range cfgs {
		grpSize, err := strconv.ParseInt(row[1], 10, 64)
		if err != nil {
			log.Fatalln(`parsing group size failed`, err)
		}

		consistnt, err := strconv.ParseBool(row[3])
		if err != nil {
			log.Fatalln(`parsing consistency failed`, err)
		}

		ordered, err := strconv.ParseBool(row[4])
		if err != nil {
			log.Fatalln(`parsing ordering failed`, err)
		}

		buf, err := strconv.ParseInt(row[5], 10, 64)
		if err != nil {
			log.Fatalln(`parsing zmq buf failed`, err)
		}

		gc = groupCfg{
			name:          row[0],
			initSize:      grpSize,
			mode:          row[2],
			consistntJoin: consistnt,
			ordered:       ordered,
			zmqBuf:        buf,
		}
	}

	// read individual members
	f2, err := os.Open(`../deployer/started_nodes.csv`)
	if err != nil {
		log.Fatal(`opening group members failed`, err)
	}
	defer f2.Close()

	records, err := csv.NewReader(f2).ReadAll()
	if err != nil {
		log.Fatal(`reading group members failed`, err)
	}

	for i, row := range records {
		if i == int(gc.initSize) {
			break
		}
		group = append(group, member{name: row[0], mockEndpoint: `http://` + row[1]})
	}

	fmt.Printf("Group initialized: %v\n", group)
	return gc
}
