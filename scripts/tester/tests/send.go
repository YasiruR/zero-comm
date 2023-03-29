package tests

import (
	"bytes"
	"encoding/json"
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

func Send(testBuf int64, usr, keyPath string, manualSize int) {
	numTests = 3
	testLatencyBuf = time.Duration(testBuf)
	if manualSize != 0 {
		fmt.Printf("\n[single-queue mode, size=%d] \n", manualSize)
		initSendTest(`sq-c-o-topic`, `single-queue`, true, true, true, int64(manualSize), usr, keyPath)
		fmt.Printf("\n[multiple-queue mode, size=%d] \n", manualSize)
		initSendTest(`mq-c-o-topic`, `multiple-queue`, true, true, true, int64(manualSize), usr, keyPath)
		return
	}

	for _, size := range latncygrpSizes {
		fmt.Printf("\n[single-queue mode, size=%d] \n", size)
		initSendTest(`sq-c-o-topic`, `single-queue`, true, true, false, int64(size), usr, keyPath)
		fmt.Printf("\n[multiple-queue mode, size=%d] \n", size)
		initSendTest(`mq-c-o-topic`, `multiple-queue`, true, true, false, int64(size), usr, keyPath)
	}
}

func initSendTest(topic, mode string, consistntJoin, ordrd, manualInit bool, size int64, usr, keyPath string) {
	cfg := group.Config{
		Topic:            topic,
		InitSize:         size,
		Mode:             mode,
		ConsistntJoin:    consistntJoin,
		Ordered:          ordrd,
		InitConnectedAll: false,
	}

	grp := group.InitGroup(cfg, testLatencyBuf, usr, keyPath, manualInit)
	time.Sleep(testLatencyBuf * time.Second)

	contList := initTestAgents(0, 1, grp, false)
	if len(contList) != 1 {
		log.Fatal(`test agent init failed`)
	}
	tester := contList[0]

	if err := tester.PubSub.Join(topic, grp[0].Name, true); err != nil {
		log.Error(fmt.Sprintf(`join failed for %s`, tester.Cfg.Name), err)
	}

	var batchSizes []int
	var latList []float64
	var pingList []int64
	fmt.Println("# Test debug logs (publish):")
	for _, bs := range publishBatchSizes {
		lats, pings := send(cfg.Topic, tester, grp, bs)
		for i, lat := range lats {
			latList = append(latList, lat)
			pingList = append(pingList, pings[i])
			batchSizes = append(batchSizes, bs)
		}
		fmt.Printf("# Average publish-latency [batch=%d]: %f ms, maximum ack latency: %f\n", bs, avg(lats), maxAckLatency(pings))
	}

	writer.Persist(`publish-latency`, cfg, batchSizes, latList, pingList)
	group.Purge(manualInit)
}

func send(topic string, tester *container.Container, grp []group.Member, msgCount int) (latList []float64, pingList []int64) {
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

				wg.Done()
				res.Body.Close()
			}(m, wg)
		}

		time.Sleep(testLatencyBuf * time.Second)
		start := time.Now()
		for j := 0; j < msgCount; j++ {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				if err = tester.PubSub.Send(topic, req.Msg); err != nil {
					log.Fatal(fmt.Sprintf(`publish error - %v`, err))
				}
			}(wg)
		}

		wg.Wait()
		lat := time.Since(start).Milliseconds()
		latList = append(latList, float64(lat))

		pingLatency, pingCount := pingAll(grp)
		if int(pingCount) != len(grp) {
			pingLatency = 0
		}
		pingList = append(pingList, pingLatency)
		fmt.Printf("	Batch-size=%d, Attempt %d: %d ms [ping-all one-round trip: %d ms]\n", msgCount, i+1, lat, pingLatency)
		time.Sleep(testLatencyBuf / 2 * time.Second)
	}

	agentPort += numTests
	pubPort += numTests
	return latList, pingList
}
