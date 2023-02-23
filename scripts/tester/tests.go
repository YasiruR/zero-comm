package main

import (
	"bytes"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"log"
	"net/http"
	"time"
)

const (
	numTests = 3
)

func joinLatency(buf int, singleQ, pub bool) {
	var total int64
	for i := 0; i < numTests; i++ {
		// init agent
		c := initAgent(fmt.Sprintf(`tester-%d`, i), 6140+i, 6240+i, buf, singleQ)
		fmt.Printf("Tester agent initialized (name: %s, port: %d, pub-endpoint: %s)\n", c.Cfg.Name, c.Cfg.Port, c.Cfg.PubEndpoint)

		go listen(c)
		go func(c *domain.Container) {
			if err := c.Server.Start(); err != nil {
				c.Log.Fatal(`tester `, `failed to start the server`, err)
			}
		}(c)

		// generate inv
		url, err := c.Prober.Invite()
		if err != nil {
			log.Fatal(`tester `, fmt.Sprintf(`failed generating inv - %s`, err))
		}

		// send inv to oob endpoint
		if _, err = http.DefaultClient.Post(group[0].mockEndpoint+mock.ConnectEndpoint, `application/octet-stream`, bytes.NewBufferString(url)); err != nil {
			log.Fatal(`tester `, err)
		}

		// start measuring time
		start := time.Now()

		// connect to group
		if err = c.PubSub.Join(group[0].topic, group[0].name, pub); err != nil {
			log.Fatal(`tester `, err)
		}

		elapsed := time.Since(start).Milliseconds()
		fmt.Printf("Attempt %d: %d ms\n", i, elapsed)
		total += elapsed

		if err = c.PubSub.Leave(group[0].topic); err != nil {
			log.Fatalln(`tester `, err)
		}
	}

	fmt.Println(`Average join-latency (ms): `, float64(total)/numTests)
}
