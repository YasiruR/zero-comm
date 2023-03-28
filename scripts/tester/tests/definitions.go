package tests

type TestMode string

const (
	JoinLatency    TestMode = `join-latency`
	JoinThroughput TestMode = `join-throughput`
)
