package main

import (
	"bytes"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"log"
	"net/http"
	"time"
)

func joinLatency(buf int, singleQ, pub bool) {
	// init agent
	c := initAgent(`tester`, 6143, 6145, buf, singleQ)
	fmt.Println(`tester agent initialized`, c.Cfg.InvEndpoint)

	go func(c *domain.Container) {
		if err := c.Server.Start(); err != nil {
			c.Log.Fatal(`tester `, `failed to start the server`, err)
		}
	}(c)

	// generate inv
	url, err := c.Prober.Invite()
	if err != nil {
		log.Fatal(`tester`, fmt.Sprintf(`failed generating inv - %s`, err))
	}

	fmt.Println("INV: ", url)

	// send inv to oob endpoint - new
	if _, err = http.DefaultClient.Post(group[0].mockEndpoint+domain.OOBEndpoint, `application/octet-stream`, bytes.NewBufferString(url)); err != nil {
		log.Fatal(`tester `, err)
	}

	fmt.Println(`oob req done`)
	//time.Sleep(2 * time.Second)

	// start measuring time
	start := time.Now()

	// connect to group
	if err = c.PubSub.Join(group[0].topic, group[0].name, pub); err != nil {
		log.Fatal(`tester `, err)
	}

	fmt.Println(`connected to group`, group[0].topic)

	// measure latency when func returns
	latency := time.Since(start)
	fmt.Printf("join latency (ms) with %s: %v\n", group[0], latency)
}
