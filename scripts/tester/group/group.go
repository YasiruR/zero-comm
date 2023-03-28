package group

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type Config struct {
	Topic            string
	InitSize         int64
	Mode             string
	ConsistntJoin    bool
	Ordered          bool
	InitConnectedAll bool // true if tester is initially connected to all members
}

type Member struct {
	Name         string
	MockEndpoint string
}

func InitGroup(cfg Config, testBuf time.Duration, usr, keyPath string, manual bool) []Member {
	if manual {
		return membrs()
	}

	var consistncy string
	if cfg.ConsistntJoin {
		consistncy = `consistent_join`
	} else {
		consistncy = `inconsistent_join`
	}

	var ordr string
	if cfg.Ordered {
		ordr = `ordered`
	} else {
		ordr = `not_ordered`
	}

	initCmd := exec.Command(`/bin/bash`, `init.sh`, strconv.FormatInt(cfg.InitSize, 10), usr, keyPath)
	initOut, err := initCmd.CombinedOutput()
	if err != nil {
		log.Fatalln(`initializing members failed -`, err, string(initOut))
	}

	time.Sleep(testBuf * time.Second)
	createCmd := exec.Command(`/bin/bash`, `create.sh`, cfg.Topic, cfg.Mode, consistncy, ordr)
	out, err := createCmd.CombinedOutput()
	if err != nil {
		log.Fatalln(`creating group failed -`, err, string(out))
	}

	return membrs()
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
