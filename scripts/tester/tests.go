package main

import (
	"bytes"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"net/http"
	"time"
)

const (
	numTests       = 3
	firstAgentPort = 6140
	firstPubPort   = 6240
)

func joinLatency(buf int, pub bool) {
	var total int64
	for i := 0; i < numTests; i++ {
		// init agent
		c := initAgent(fmt.Sprintf(`tester-%d`, i+1), firstAgentPort+i, firstPubPort+i, buf)
		fmt.Printf("Tester agent initialized (name: %s, port: %d, pub-endpoint: %s)\n", c.Cfg.Name, c.Cfg.Port, c.Cfg.PubEndpoint)

		go listen(c)
		go func(c *container.Container) {
			if err := c.Server.Start(); err != nil {
				c.Log.Fatal(`tester`, `failed to start the server`, err)
			}
		}(c)

		// generate inv
		url, err := c.Prober.Invite()
		if err != nil {
			c.Log.Fatal(`tester`, fmt.Sprintf(`failed generating inv - %s`, err))
		}

		// send inv to oob endpoint
		if _, err = http.DefaultClient.Post(group[0].mockEndpoint+mock.ConnectEndpoint, `application/octet-stream`, bytes.NewBufferString(url)); err != nil {
			c.Log.Fatal(`tester`, err)
		}

		// start measuring time
		start := time.Now()

		// connect to group
		if err = c.PubSub.Join(group[0].topic, group[0].name, pub); err != nil {
			c.Log.Fatal(`tester`, err)
		}

		elapsed := time.Since(start).Milliseconds()
		fmt.Printf("Attempt %d: %d ms\n\n", i+1, elapsed)
		total += elapsed

		if err = c.PubSub.Leave(group[0].topic); err != nil {
			c.Log.Fatal(`tester`, err)
		}
	}

	fmt.Printf("Average join-latency (ms): %f\n", float64(total)/numTests)
	fmt.Println(`----- END -----`)
}
