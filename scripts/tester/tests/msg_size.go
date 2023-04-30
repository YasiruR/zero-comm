package tests

import (
	"fmt"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"github.com/tryfix/log"
	"time"
)

func Size() {
	topic := `test`
	cfg := group.Config{
		Topic:            topic,
		InitSize:         1,
		Mode:             `single-queue`,
		ConsistntJoin:    false,
		Ordered:          false,
		InitConnectedAll: false,
	}

	grp := group.InitGroup(cfg, testLatencyBuf, ``, ``, false)
	time.Sleep(testLatencyBuf * time.Second)
	fmt.Println("# Test debug logs (publish):")

	contList := initTestAgents(0, 1, grp, false)
	if len(contList) != 1 {
		log.Fatal(`test agent init failed`)
	}
	tester := contList[0]

	if err := tester.PubSub.Join(topic, grp[0].Name, true); err != nil {
		log.Error(fmt.Sprintf(`join failed for %s`, tester.Cfg.Name), err)
	}

	var initSizes []int
	var didcommSizes []int
	for _, msg := range testMsgs {
		initSizes = append(initSizes, len([]byte(msg)))
		n, err := tester.PubSub.Send(topic, msg)
		if err != nil || len(n) == 0 {
			log.Fatal(fmt.Sprintf(`sending test message (n=%v) failed - %v`, n, err))
			time.Sleep(testLatencyBuf * time.Second)
		}
		didcommSizes = append(didcommSizes, n[0])
	}

	fmt.Println("	Initial message sizes: ", initSizes)
	fmt.Println("	DIDComm message sizes: ", didcommSizes)
}
