package writer

import (
	"encoding/csv"
	"fmt"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"log"
	"os"
	"strconv"
)

// file names
const (
	joinLatFile = `results/join_latency.csv`
	joinThrFile = `results/join_throughput.csv`
	pubLatFile  = `results/publish_latency.csv`
)

func Persist(test string, gc group.Config, batchSizes []int, results []float64, pingLat []int64, SuccsList ...float64) {
	switch test {
	case `join-latency`:
		f, w := readFile(joinLatFile, []string{`name`, `initial_size`, `mode`, `consistent_join`, `causally_ordered`, `init_connected`, `latency_ms`})
		writeJoinLatency(f, w, gc, results)
	case `join-throughput`:
		f, w := readFile(joinThrFile, []string{`name`, `initial_size`, `mode`, `consistent_join`, `causally_ordered`, `init_connected`, `batch_size`, `latency_ms`})
		writeJoinThroughput(f, w, gc, batchSizes, results)
	case `publish-latency`:
		f, w := readFile(pubLatFile, []string{`name`, `initial_size`, `batch_size`, `ping_ms`, `success_rate`, `latency_ms`})
		writePublishLatency(f, w, gc, batchSizes, pingLat, SuccsList, results)
	default:
		fmt.Printf("# Skipped persisting results (test=%s)\n", test)
	}
}

func readFile(path string, headers []string) (*os.File, *csv.Writer) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln(`opening or creating join.csv failed -`, err)
	}

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

	return file, w
}

func writeJoinThroughput(f *os.File, w *csv.Writer, gc group.Config, batchSizes []int, results []float64) {
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
	f.Close()
}

func writeJoinLatency(f *os.File, w *csv.Writer, gc group.Config, results []float64) {
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
	f.Close()
}

func writePublishLatency(f *os.File, w *csv.Writer, gc group.Config, batchSizes []int, pingLat []int64, succsList, results []float64) {
	for i, r := range results {
		var pl int64
		if pingLat != nil {
			pl = pingLat[i]
		}
		if err := w.Write([]string{
			gc.Topic,
			strconv.FormatInt(gc.InitSize, 10),
			fmt.Sprintf(`%d`, batchSizes[i]),
			fmt.Sprintf(`%d`, pl),
			fmt.Sprintf(`%f`, succsList[i]),
			fmt.Sprintf(`%f`, r),
		}); err != nil {
			log.Fatalln(`writing join results failed -`, err)
		}
	}
	w.Flush()
	f.Close()
}
