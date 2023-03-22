package writer

import (
	"encoding/csv"
	"fmt"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"log"
	"os"
	"strconv"
)

func Persist(test string, gc group.Config, batchSizes []int, results []float64) {
	switch test {
	case `join-latency`:
		writeJoinLatency(
			readFile(`../tester/results/join_latency.csv`, []string{`name`, `initial_size`, `mode`, `consistent_join`, `causally_ordered`, `init_connected`, `latency_ms`}),
			gc,
			results)
	case `join-throughput`:
		writeJoinThroughput(
			readFile(`../tester/results/join_throughput.csv`, []string{`name`, `initial_size`, `mode`, `consistent_join`, `causally_ordered`, `init_connected`, `batch_size`, `latency_ms`}),
			gc,
			batchSizes,
			results)
	}
}

func readFile(path string, headers []string) *csv.Writer {
	//const path = `../tester/results/join.csv`
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln(`opening or creating join.csv failed -`, err)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		log.Fatalln(`reading existing join results failed -`, err)
	}

	if len(rows) == 0 {
		if err = w.Write(headers); err != nil {
			log.Fatalln(`writing join headers failed -`, err)
		}
	}

	return w
}

func writeJoinThroughput(w *csv.Writer, gc group.Config, batchSizes []int, results []float64) {
	for i, r := range results {
		if err := w.Write([]string{
			gc.Topic,
			strconv.FormatInt(gc.InitSize, 10),
			gc.Mode,
			fmt.Sprintf(`%t`, gc.ConsistntJoin),
			fmt.Sprintf(`%t`, gc.Ordered),
			fmt.Sprintf(`%t`, gc.InitConnectedAll),
			fmt.Sprintf(`%d`, batchSizes[i]),
			fmt.Sprintf(`%f`, r),
		}); err != nil {
			log.Fatalln(`writing join results failed -`, err)
		}
	}
	w.Flush()
}

func writeJoinLatency(w *csv.Writer, gc group.Config, results []float64) {
	for _, r := range results {
		if err := w.Write([]string{
			gc.Topic,
			strconv.FormatInt(gc.InitSize, 10),
			gc.Mode,
			fmt.Sprintf(`%t`, gc.ConsistntJoin),
			fmt.Sprintf(`%t`, gc.Ordered),
			fmt.Sprintf(`%t`, gc.InitConnectedAll),
			fmt.Sprintf(`%f`, r),
		}); err != nil {
			log.Fatalln(`writing join results failed -`, err)
		}
	}
	w.Flush()
}
