package group

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

type Config struct {
	Topic         string
	InitSize      int64
	Mode          string
	ConsistntJoin bool
	Ordered       bool
	ZmqBuf        int64
}

type Member struct {
	Name         string
	MockEndpoint string
}

func InitGroup(topic, mode string, consistntJoin, ordrd bool, size, zmqBuf int64) (Config, []Member) {
	var consistncy string
	if consistntJoin {
		consistncy = `consistent_join`
	} else {
		consistncy = `inconsistent_join`
	}

	var ordr string
	if ordrd {
		ordr = `ordered`
	} else {
		ordr = `not_ordered`
	}

	initCmd := exec.Command(`/bin/bash`, `init.sh`, strconv.FormatInt(size, 10), strconv.FormatInt(zmqBuf, 10))
	initOut, err := initCmd.CombinedOutput()
	if err != nil {
		log.Fatalln(`initializing members failed -`, err, string(initOut))
	}

	cmd := exec.Command(`/bin/bash`, `create.sh`, topic, mode, consistncy, ordr, strconv.FormatInt(size, 10), strconv.FormatInt(zmqBuf, 10))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalln(`creating group failed -`, err, string(out))
	}

	// todo check why inconsistency happens

	return Config{
		Topic:         topic,
		InitSize:      size,
		Mode:          mode,
		ConsistntJoin: consistntJoin,
		Ordered:       ordrd,
		ZmqBuf:        zmqBuf,
	}, membrs()
}

func membrs() []Member {
	// read individual members
	f2, err := os.Open(`started_nodes.csv`)
	if err != nil {
		log.Fatalln(`opening group members failed -`, err)
	}
	defer f2.Close()

	records, err := csv.NewReader(f2).ReadAll()
	if err != nil {
		log.Fatalln(`reading group members failed -`, err)
	}

	var group []Member
	for _, row := range records {
		group = append(group, Member{Name: row[0], MockEndpoint: `http://` + row[1]})
	}

	fmt.Printf("# Group initialized: %v\n", group)
	return group
}

func Purge() {
	cmd := exec.Command(`/bin/bash`, `term.sh`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalln(`purging group failed -`, err, string(out))
	}
}
