package tests

import (
	"bytes"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"github.com/YasiruR/didcomm-prober/scripts/tester/writer"
	"net/http"
	"sync"
	"time"
)

const (
	numTests = 3
)

var (
	//groupSizes     = []int{1, 2, 5, 10, 20, 50, 100}
	groupSizes                   = []int{1, 2, 5, 10}
	firstAgentPort               = 6140
	firstPubPort                 = 6540
	testLatencyBuf time.Duration = 0
)

// todo test joining with multiple groups

func Join(testBuf int64, usr, keyPath string) {
	testLatencyBuf = time.Duration(testBuf)
	for _, size := range groupSizes {
		conctd := false
		for i := 0; i < 2; i++ {
			fmt.Printf("\n[single-queue, join-consistent, ordered, size=%d connected=%t] \n", size, conctd)
			joinTest(`sq-c-o-topic`, `single-queue`, true, true, conctd, int64(size), usr, keyPath)

			//fmt.Printf("\n[multiple-queue, join-consistent, ordered, size=%d connected=%t] \n", size, conctd)
			//joinTest(`mq-c-o-topic`, `multiple-queue`, true, true, conctd, int64(size), usr, keyPath)
			//
			//fmt.Printf("\n[single-queue, join-inconsistent, ordered, size=%d connected=%t] \n", size, conctd)
			//joinTest(`sq-i-o-topic`, `single-queue`, false, true, conctd, int64(size), usr, keyPath)
			//
			//fmt.Printf("\n[multiple-queue, join-inconsistent, ordered, size=%d connected=%t] \n", size, conctd)
			//joinTest(`mq-i-o-topic`, `multiple-queue`, false, true, conctd, int64(size), usr, keyPath)
			//
			//fmt.Printf("\n[single-queue, join-consistent, not-ordered, size=%d connected=%t] \n", size, conctd)
			//joinTest(`sq-c-no-topic`, `single-queue`, true, false, conctd, int64(size), usr, keyPath)
			//
			//fmt.Printf("\n[multiple-queue, join-consistent, not-ordered, size=%d connected=%t] \n", size, conctd)
			//joinTest(`mq-c-no-topic`, `multiple-queue`, true, false, conctd, int64(size), usr, keyPath)
			//
			//fmt.Printf("\n[single-queue, join-inconsistent, not-ordered, size=%d connected=%t] \n", size, conctd)
			//joinTest(`sq-i-no-topic`, `single-queue`, false, false, conctd, int64(size), usr, keyPath)
			//
			//fmt.Printf("\n[multiple-queue, join-inconsistent, not-ordered, size=%d connected=%t] \n", size, conctd)
			//joinTest(`mq-i-no-topic`, `multiple-queue`, false, false, conctd, int64(size), usr, keyPath)
			conctd = true
		}
	}
}

func joinTest(topic, mode string, consistntJoin, ordrd, conctd bool, size int64, usr, keyPath string) {
	cfg := group.Config{
		Topic:            topic,
		InitSize:         size,
		Mode:             mode,
		ConsistntJoin:    consistntJoin,
		Ordered:          ordrd,
		InitConnectedAll: conctd,
	}

	grp := group.InitGroup(cfg, testLatencyBuf, usr, keyPath)
	time.Sleep(testLatencyBuf * time.Second)

	fmt.Println("# Test debug logs:")
	latList := join(cfg.Topic, true, conctd, grp)
	writer.Persist(`join`, cfg, latList)

	fmt.Printf("# Average join-latency (ms): %f\n", avg(latList))
	group.Purge()
}

func join(topic string, pub, conctd bool, grp []group.Member) (latList []float64) {
	for i := 0; i < numTests; i++ {
		// init agent
		c := group.InitAgent(fmt.Sprintf(`tester-%d`, i+1), firstAgentPort+i, firstPubPort+i)
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

		wg := &sync.WaitGroup{}
		for j, m := range grp {
			if !conctd && j == 1 {
				break
			}

			wg.Add(1)
			go func(m group.Member) {
				if _, err = http.DefaultClient.Post(m.MockEndpoint+mock.ConnectEndpoint, `application/octet-stream`, bytes.NewBufferString(url)); err != nil {
					c.Log.Fatal(err)
				}
				wg.Done()
			}(m)
		}

		wg.Wait()
		time.Sleep(testLatencyBuf * time.Second)

		// start measuring time
		start := time.Now()

		// connect to group
		if err = c.PubSub.Join(topic, grp[0].Name, pub); err != nil {
			c.Log.Fatal(err)
		}

		elapsed := time.Since(start).Milliseconds()
		fmt.Printf("	Attempt %d: %d ms\n", i+1, elapsed)
		latList = append(latList, float64(elapsed))

		if err = c.PubSub.Leave(topic); err != nil {
			c.Log.Fatal(err)
		}

		time.Sleep(testLatencyBuf * time.Second)
	}

	// can remove by making constant
	firstAgentPort += numTests
	firstPubPort += numTests
	return latList
}

func avg(latList []float64) float64 {
	var total float64
	for _, l := range latList {
		total += l
	}

	return total / float64(len(latList))
}
