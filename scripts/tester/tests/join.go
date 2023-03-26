package tests

import (
	"bytes"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"github.com/YasiruR/didcomm-prober/scripts/tester/writer"
	"github.com/tryfix/log"
	"net/http"
	"sync"
	"time"
)

var (
	numTests int
	//groupSizes     = []int{1, 2, 5, 10, 20, 50, 100}
	groupSizes                   = []int{2, 5, 10, 20, 25}
	testBatchSizes               = []int{5, 10}
	agentPort                    = 6140
	pubPort                      = 6540
	testLatencyBuf time.Duration = 0
)

// todo test joining with multiple groups

func Join(testBuf int64, usr, keyPath string) {
	testLatencyBuf = time.Duration(testBuf)
	for _, size := range groupSizes {
		conctd := true
		for i := 0; i < 2; i++ {
			// when initial group size is 1, connected-to-all will have no impact
			if size == 1 && i == 1 {
				continue
			}

			fmt.Printf("\n[single-queue, join-consistent, ordered, size=%d connected=%t] \n", size, conctd)
			initJoinTest(`sq-c-o-topic`, `single-queue`, true, true, conctd, int64(size), usr, keyPath)

			//fmt.Printf("\n[multiple-queue, join-consistent, ordered, size=%d connected=%t] \n", size, conctd)
			//initJoinTest(`mq-c-o-topic`, `multiple-queue`, true, true, conctd, int64(size), usr, keyPath)

			conctd = false
		}
	}
}

func initJoinTest(topic, mode string, consistntJoin, ordrd, conctd bool, size int64, usr, keyPath string) {
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

	fmt.Println("# Test debug logs (latency):")
	numTests = 3
	latList := join(cfg.Topic, true, conctd, grp, 1)
	writer.Persist(`join-latency`, cfg, nil, latList)

	//numTests = 1
	//var thrLatList []float64
	//for _, bs := range testBatchSizes {
	//	fmt.Printf("# Test debug logs (throughput) [batch-size=%d]:\n", bs)
	//	thrLatList = append(thrLatList, join(cfg.Topic, true, conctd, grp, bs)[0])
	//}
	//writer.Persist(`join-throughput`, cfg, testBatchSizes, thrLatList)

	fmt.Printf("# Average join-latency (ms): %f\n", avg(latList))
	//out := `# Load test results [batch-size, latency(ms)]: `
	//for i, lat := range thrLatList {
	//	out += fmt.Sprintf(`%d:%f `, testBatchSizes[i], lat)
	//}
	group.Purge()
}

func join(topic string, pub, conctd bool, grp []group.Member, count int) (latList []float64) {
	for i := 0; i < numTests; i++ {
		// init agents
		var contList []*container.Container
		for k := 0; k < count; k++ {
			c := group.InitAgent(fmt.Sprintf(`tester-%d`, (count*i)+k+1), agentPort+(count*i)+k, pubPort+(count*i)+k)
			fmt.Printf("	Tester agent initialized (name: %s, port: %d, pub-endpoint: %s)\n", c.Cfg.Name, c.Cfg.Port, c.Cfg.PubEndpoint)
			contList = append(contList, c)

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
				go func(m group.Member, wg *sync.WaitGroup) {
					if _, err = http.DefaultClient.Post(m.MockEndpoint+mock.ConnectEndpoint, `application/octet-stream`, bytes.NewBufferString(url)); err != nil {
						log.Fatal(err)
					}
					wg.Done()
				}(m, wg)
			}
			wg.Wait()
		}

		time.Sleep(testLatencyBuf * time.Second)
		wg := &sync.WaitGroup{}
		accptrs := acceptors(count, len(grp), conctd)

		start := time.Now()
		for j, c := range contList {
			wg.Add(1)
			go func(accptrId int, c *container.Container, wg *sync.WaitGroup) {
				if err := c.PubSub.Join(topic, grp[accptrId].Name, pub); err != nil {
					log.Error(fmt.Sprintf(`join failed for %s`, c.Cfg.Name), err)
				}
				wg.Done()
			}(accptrs[j], c, wg)
		}

		wg.Wait()
		latency := time.Since(start).Milliseconds()
		fmt.Printf("	Attempt %d: %d ms\n", i+1, latency)

		// check if group has correct #members?

		for _, c := range contList {
			if err := c.PubSub.Leave(topic); err != nil {
				log.Error(fmt.Sprintf(`leaving group failed for %s`, c.Cfg.Name), err)
			}
		}

		latList = append(latList, float64(latency))
		time.Sleep(testLatencyBuf * time.Second)
	}

	agentPort += count * numTests
	pubPort += count * numTests
	return latList
}

func avg(latList []float64) float64 {
	var total float64
	for _, l := range latList {
		total += l
	}

	return total / float64(len(latList))
}

func acceptors(joinrCount, grpSize int, conctd bool) (ids []int) {
	for joinrId := 0; joinrId < joinrCount; joinrId++ {
		// send join requests to different members if joiner is connected to all
		if conctd && joinrCount > 1 {
			// changing acceptor based on the joiner
			accptrId := joinrId
			if joinrId > grpSize-1 {
				accptrId = joinrId % grpSize
			}

			ids = append(ids, accptrId)
		} else {
			ids = append(ids, 0)
		}
	}

	return ids
}
