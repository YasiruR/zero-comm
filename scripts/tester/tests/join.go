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
	numTests       = 3
	testLatencyBuf = 2
)

var (
	//groupSizes     = []int{1, 2, 5, 10, 20, 50, 100}
	groupSizes     = []int{1, 2}
	firstAgentPort = 6140
	firstPubPort   = 6540
)

// todo test joining with multiple groups

func Join(buf int64) {
	for _, size := range groupSizes {
		fmt.Printf("\n[single-queue, join-consistent, ordered, size=%d, buffer=%d] \n", size, buf)
		joinTest(`sq-c-o-topic`, `single-queue`, true, true, int64(size), buf)

		//fmt.Printf("\n[multiple-queue, join-consistent, ordered, size=%d, buffer=%d] \n", size, buf)
		//joinTest(`mq-c-o-topic`, `multiple-queue`, true, true, int64(size), buf)
		//
		//fmt.Printf("\n[single-queue, join-inconsistent, ordered, size=%d, buffer=%d] \n", size, buf)
		//joinTest(`sq-i-o-topic`, `single-queue`, false, true, int64(size), buf)
		//
		//fmt.Printf("\n[multiple-queue, join-inconsistent, ordered, size=%d, buffer=%d] \n", size, buf)
		//joinTest(`mq-i-o-topic`, `multiple-queue`, false, true, int64(size), buf)
		//
		//fmt.Printf("\n[single-queue, join-consistent, not-ordered, size=%d, buffer=%d] \n", size, buf)
		//joinTest(`sq-c-no-topic`, `single-queue`, true, false, int64(size), buf)
		//
		//fmt.Printf("\n[multiple-queue, join-consistent, not-ordered, size=%d, buffer=%d] \n", size, buf)
		//joinTest(`mq-c-no-topic`, `multiple-queue`, true, false, int64(size), buf)
		//
		//fmt.Printf("\n[single-queue, join-inconsistent, not-ordered, size=%d, buffer=%d] \n", size, buf)
		//joinTest(`sq-i-no-topic`, `single-queue`, false, false, int64(size), buf)
		//
		//fmt.Printf("\n[multiple-queue, join-inconsistent, not-ordered, size=%d, buffer=%d] \n", size, buf)
		//joinTest(`mq-i-no-topic`, `multiple-queue`, false, false, int64(size), buf)
	}
}

func joinTest(topic, mode string, consistntJoin, ordrd bool, size, zmqBuf int64) {
	cfg, grp := group.InitGroup(topic, mode, consistntJoin, ordrd, size, zmqBuf)
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
				c.Log.Fatal(`failed to start the server`, err)
			}
		}(c)

		// generate inv
		url, err := c.Prober.Invite()
		if err != nil {
			c.Log.Fatal(fmt.Sprintf(`failed generating inv - %s`, err))
		}

		fmt.Println("INV GEND", grp[0].MockEndpoint+mock.ConnectEndpoint)
		fmt.Println("INV: ", url)
		fmt.Println()

		// send inv to oob endpoint
		if _, err = http.DefaultClient.Post(grp[0].MockEndpoint+mock.ConnectEndpoint, `application/octet-stream`, bytes.NewBufferString(url)); err != nil {
			c.Log.Fatal(err)
		}

		fmt.Println("INV SENT", grp[0].MockEndpoint+mock.ConnectEndpoint)

		time.Sleep(testLatencyBuf * time.Second)

		// start measuring time
		start := time.Now()

		fmt.Println("JOIN INITING")

		// connect to group
		if err = c.PubSub.Join(topic, grp[0].Name, pub); err != nil {
			c.Log.Fatal(err)
		}

		fmt.Println("JOIN DONE")

		elapsed := time.Since(start).Milliseconds()
		fmt.Printf("	Attempt %d: %d ms\n", i+1, elapsed)
		total += elapsed

		if err = c.PubSub.Leave(topic); err != nil {
			c.Log.Fatal(err)
		}

		time.Sleep(testLatencyBuf * time.Second)
	}

	// can remove by making constant
	firstAgentPort += numTests
	firstPubPort += numTests
	return float64(total) / numTests
}
