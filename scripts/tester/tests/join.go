package tests

import (
	"bytes"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"github.com/YasiruR/didcomm-prober/scripts/tester/writer"
	"net/http"
	"time"
)

const (
	numTests = 3
	//firstAgentPort = 6140
	//firstPubPort   = 6240
)

var (
	//groupSizes     = []int{1, 2, 5, 10, 20}
	groupSizes     = []int{5}
	firstAgentPort = 6140
	firstPubPort   = 6240
)

// todo test joining with multiple groups

func Join(buf int) {
	for _, size := range groupSizes {
		joinTest(`sq-c-o-topic`, `single-queue`, true, true, int64(size), int64(buf))
		//joinTest(`mq-c-o-topic`, `multiple-queue`, true, true, int64(size), int64(buf))
		//joinTest(`sq-i-o-topic`, `single-queue`, false, true, int64(size), int64(buf))
		//joinTest(`mq-i-o-topic`, `multiple-queue`, false, true, int64(size), int64(buf))
		//joinTest(`sq-c-no-topic`, `single-queue`, true, false, int64(size), int64(buf))
		//joinTest(`mq-c-no-topic`, `multiple-queue`, true, false, int64(size), int64(buf))
		//joinTest(`sq-i-no-topic`, `single-queue`, false, false, int64(size), int64(buf))
		//joinTest(`mq-i-no-topic`, `multiple-queue`, false, false, int64(size), int64(buf))
	}
}

func joinTest(topic, mode string, consistntJoin, ordrd bool, size, zmqBuf int64) {
	cfg, grp := group.InitGroup(topic, mode, consistntJoin, ordrd, size, zmqBuf)
	fmt.Printf("# Join latency test for an initial group size of %d with %dms zmq buffer\n", cfg.InitSize, cfg.ZmqBuf)
	fmt.Println("# Test debug logs:")

	avg := join(cfg.Topic, int(zmqBuf), true, grp)
	writer.Persist(`join`, cfg, []float64{avg})
	fmt.Printf("# Average join-latency (ms): %f\n", avg)
	group.Purge()
}

func join(topic string, buf int, pub bool, grp []group.Member) float64 {
	var total int64
	for i := 0; i < numTests; i++ {
		// init agent
		c := group.InitAgent(fmt.Sprintf(`tester-%d`, i+1), firstAgentPort+i, firstPubPort+i, buf)
		fmt.Printf("	Tester agent initialized (name: %s, port: %d, pub-endpoint: %s)\n", c.Cfg.Name, c.Cfg.Port, c.Cfg.PubEndpoint)

		go group.Listen(c)
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
		if _, err = http.DefaultClient.Post(grp[0].MockEndpoint+mock.ConnectEndpoint, `application/octet-stream`, bytes.NewBufferString(url)); err != nil {
			c.Log.Fatal(`tester`, err)
		}

		// start measuring time
		start := time.Now()

		// connect to group
		if err = c.PubSub.Join(topic, grp[0].Name, pub); err != nil {
			c.Log.Fatal(`tester`, err)
		}

		elapsed := time.Since(start).Milliseconds()
		fmt.Printf("	Attempt %d: %d ms\n", i+1, elapsed)
		total += elapsed

		if err = c.PubSub.Leave(topic); err != nil {
			c.Log.Fatal(`tester`, err)
		}
	}

	// can remove by making constant
	firstAgentPort += numTests
	firstPubPort += numTests
	return float64(total) / numTests
}
