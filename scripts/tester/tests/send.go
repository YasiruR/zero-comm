package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"github.com/YasiruR/didcomm-prober/scripts/tester/writer"
	"github.com/tryfix/log"
	"net/http"
	"sync"
	"time"
)

func Send(testBuf int64, usr, keyPath string) {
	numTests = 3
	testLatencyBuf = time.Duration(testBuf)
	for _, size := range latncygrpSizes {
		fmt.Printf("\n[single-queue, join-consistent, ordered, size=%d] \n", size)
		initSendTest(`sq-c-o-topic`, `single-queue`, true, true, int64(size), usr, keyPath)
	}
}

func initSendTest(topic, mode string, consistntJoin, ordrd bool, size int64, usr, keyPath string) {
	cfg := group.Config{
		Topic:            topic,
		InitSize:         size,
		Mode:             mode,
		ConsistntJoin:    consistntJoin,
		Ordered:          ordrd,
		InitConnectedAll: false,
	}

	grp := group.InitGroup(cfg, testLatencyBuf, usr, keyPath)
	time.Sleep(testLatencyBuf * time.Second)

	latList := send(cfg.Topic, grp, 1)
	writer.Persist(`publish-latency`, cfg, []int{1}, latList)
	fmt.Printf("# Average publish-latency (ms): %f\n", avg(latList))

	group.Purge()
}

func send(topic string, grp []group.Member, msgCount int) (latList []float64) {
	contList := initTestAgents(0, 1, grp, false)
	if len(contList) != 1 {
		log.Fatal(`test agent init failed`)
	}
	tester := contList[0]

	if err := tester.PubSub.Join(topic, grp[0].Name, true); err != nil {
		log.Error(fmt.Sprintf(`join failed for %s`, tester.Cfg.Name), err)
	}

	req := mock.ReqRegAck{Peer: tester.Cfg.Name, Msg: `t35T1n9`, Count: msgCount}
	data, err := json.Marshal(req)
	if err != nil {
		log.Fatal(fmt.Sprintf(`marshal error - %v`, err))
	}

	for i := 0; i < numTests; i++ {
		wg := &sync.WaitGroup{}
		for _, m := range grp {
			wg.Add(1)
			go func(m group.Member, wg *sync.WaitGroup) {
				res, err := http.Post(m.MockEndpoint+mock.GrpMsgAckEndpoint, `application/json`, bytes.NewReader(data))
				if err != nil {
					log.Fatal(fmt.Sprintf(`posting register ack request failed - %v`, err))
				}

				if res.StatusCode != http.StatusOK {
					res.Body.Close()
					log.Fatal(fmt.Sprintf(`registration failed (code=%d)`, res.StatusCode))
				}

				res.Body.Close()
				wg.Done()
			}(m, wg)
		}

		time.Sleep(testLatencyBuf * time.Second)
		start := time.Now()
		if err = tester.PubSub.Send(topic, req.Msg); err != nil {
			log.Fatal(fmt.Sprintf(`publish error - %v`, err))
		}

		wg.Wait()
		lat := time.Since(start).Milliseconds()
		latList = append(latList, float64(lat))
	}

	agentPort += numTests
	pubPort += numTests
	return latList
}
