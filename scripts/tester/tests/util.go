package tests

import (
	"bytes"
	"fmt"
	"github.com/YasiruR/didcomm-prober/domain/container"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"github.com/tryfix/log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func initTestAgents(testId, count int, grp []group.Member, conctd bool) []*container.Container {
	var contList []*container.Container
	for i := 0; i < count; i++ {
		agentId++
		c := group.InitAgent(fmt.Sprintf(`tester-%d`, agentId), agentPort+(count*testId)+i, pubPort+(count*testId)+i)
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

	return contList
}

func avg(list []float64) float64 {
	var total float64
	for _, l := range list {
		total += l
	}

	return total / float64(len(list))
}

func maxAckLatency(ackList []int64) float64 {
	var max float64
	for _, l := range ackList {
		if max < float64(l) {
			max = float64(l)
		}
	}
	return max
}

func pingAll(grp []group.Member) (latency, success int64) {
	wg := &sync.WaitGroup{}
	var count int64
	pingStart := time.Now()
	for _, m := range grp {
		wg.Add(1)
		go func(m group.Member, wg *sync.WaitGroup) {
			defer wg.Done()
			if _, err := http.Get(m.MockEndpoint + mock.PingEndpoint); err != nil {
				log.Error(fmt.Sprintf(`ping request to %s failed - %v`, m.MockEndpoint+mock.PingEndpoint, err))
				return
			}
			atomic.AddInt64(&count, 1)
		}(m, wg)
	}

	wg.Wait()
	return time.Since(pingStart).Milliseconds() / 2.0, count
}

func successRate(membrCount, msgCount, totalMsgs int) float64 {
	target := membrCount * msgCount
	if totalMsgs == target {
		return 100.0
	}

	return (float64(totalMsgs) / float64(target)) * 100.0
}

func pingAllMap(grp []group.Member) (pingLats map[string]int64) {
	pingSyncMap := &sync.Map{}
	wg := &sync.WaitGroup{}
	for _, m := range grp {
		wg.Add(1)
		go func(m group.Member, wg *sync.WaitGroup) {
			defer wg.Done()
			pingStart := time.Now()
			if _, err := http.Get(m.MockEndpoint + mock.PingEndpoint); err != nil {
				log.Error(fmt.Sprintf(`ping request to %s failed - %v`, m.MockEndpoint+mock.PingEndpoint, err))
				return
			}
			pingSyncMap.Store(m.Name, time.Since(pingStart).Milliseconds())
		}(m, wg)
	}

	wg.Wait()
	pingLats = map[string]int64{}
	pingSyncMap.Range(func(key, val any) bool {
		pingLats[key.(string)] = val.(int64)
		return true
	})

	return pingLats
}
