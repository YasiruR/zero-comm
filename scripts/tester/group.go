package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
)

type member struct {
	topic        string
	name         string
	mockEndpoint string
}

var group []member

func initGroup(size int) {
	// read from csv
	f, err := os.Open(`agents.csv`)
	if err != nil {
		log.Fatal(`tester`, err)
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatal(`tester`, `reading records failed`, err)
	}

	for i, row := range records {
		if i == size {
			break
		}
		group = append(group, member{topic: row[0], name: row[1], mockEndpoint: row[2]})
	}

	fmt.Printf("group initialized: %v\n", group)
}
