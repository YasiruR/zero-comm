package tests

import "time"

type TestMode string

const (
	JoinLatency    TestMode = `join-latency`
	JoinThroughput TestMode = `join-throughput`
	PublishLatency TestMode = `publish-latency`
)

var (
	numTests            int
	latncygrpSizes                    = []int{1, 2, 4, 8, 16}
	thrptGrpSizes                     = []int{4, 16, 32}
	thrptTestBatchSizes               = []int{4, 16}
	agentId                           = 0
	agentPort                         = 6140
	pubPort                           = 6540
	testLatencyBuf      time.Duration = 0
)
