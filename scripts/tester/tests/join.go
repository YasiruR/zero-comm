package tests

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"github.com/YasiruR/didcomm-prober/scripts/tester/writer"
	"github.com/tryfix/log"
	"sync"
	"time"
)

// todo test joining with multiple groups

func Join(typ TestMode, testBuf int64, usr, keyPath string, manualSize int) {
	var grpSizes []int
	if typ == JoinLatency {
		grpSizes = latncygrpSizes
	} else {
		grpSizes = thrptGrpSizes
	}

	testLatencyBuf = time.Duration(testBuf)
	if manualSize != 0 {
		fmt.Printf("\n[single-queue, join-consistent, ordered, size=%d] \n", manualSize)
		initJoinTest(typ, `sq-c-o-topic`, `single-queue`, true, true, true, int64(manualSize), usr, keyPath)
		return
	}

	for _, size := range grpSizes {
		fmt.Printf("\n[single-queue, join-consistent, ordered, size=%d] \n", size)
		initJoinTest(typ, `sq-c-o-topic`, `single-queue`, true, true, false, int64(size), usr, keyPath)
		//fmt.Printf("\n[multiple-queue, join-consistent, ordered, size=%d connected=%t] \n", size, conctd)
		//initJoinTest(typ, `mq-c-o-topic`, `multiple-queue`, true, true, false, int64(size), usr, keyPath)
	}
}

func initJoinTest(typ TestMode, topic, mode string, consistntJoin, ordrd, manualInit bool, size int64, usr, keyPath string) {
	cfg := group.Config{
		Topic:            topic,
		InitSize:         size,
		Mode:             mode,
		ConsistntJoin:    consistntJoin,
		Ordered:          ordrd,
		InitConnectedAll: true,
	}

	grp := group.InitGroup(cfg, testLatencyBuf, usr, keyPath, manualInit)
	time.Sleep(testLatencyBuf * time.Second)

	for i := 0; i < 2; i++ {
		// when initial group size is 1, connected-to-all will have no impact
		if size == 1 && i == 1 {
			continue
		}

		if typ == JoinLatency {
			joinLatency(cfg, grp)
		} else {
			joinThroughput(cfg, grp)
		}

		cfg.InitConnectedAll = false
	}

	group.Purge(manualInit)
}

func joinLatency(cfg group.Config, grp []group.Member) {
	fmt.Println("# Test debug logs (latency):")
	numTests = 3
	latList := join(cfg.Topic, true, cfg.InitConnectedAll, grp, 1)
	writer.Persist(`join-latency`, cfg, nil, latList)
	fmt.Printf("# Average join-latency (ms) [connected=%t]: %f\n", cfg.InitConnectedAll, avg(latList))
}

func joinThroughput(cfg group.Config, grp []group.Member) {
	numTests = 1
	var thrLatList []float64
	for _, bs := range thrptTestBatchSizes {
		fmt.Printf("# Test debug logs (throughput) [batch-size=%d]:\n", bs)
		thrLatList = append(thrLatList, join(cfg.Topic, true, cfg.InitConnectedAll, grp, bs)[0])
	}
	writer.Persist(`join-throughput`, cfg, thrptTestBatchSizes, thrLatList)

	out := fmt.Sprintf(`# Load test results {batch-size, latency(ms)} [connected=%t]: `, cfg.InitConnectedAll)
	for i, lat := range thrLatList {
		out += fmt.Sprintf(`%d:%f `, thrptTestBatchSizes[i], lat)
	}
	fmt.Println(out)
}

func join(topic string, pub, conctd bool, grp []group.Member, count int) (latList []float64) {
	for i := 0; i < numTests; i++ {
		contList := initTestAgents(i, count, grp, conctd)
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
