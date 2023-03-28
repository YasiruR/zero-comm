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
)

func initTestAgents(testId, count int, grp []group.Member, conctd bool) []*container.Container {
	var contList []*container.Container
	for k := 0; k < count; k++ {
		agentId++
		c := group.InitAgent(fmt.Sprintf(`tester-%d`, agentId), agentPort+(count*testId)+k, pubPort+(count*testId)+k)
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

func avg(latList []float64) float64 {
	var total float64
	for _, l := range latList {
		total += l
	}

	return total / float64(len(latList))
}
