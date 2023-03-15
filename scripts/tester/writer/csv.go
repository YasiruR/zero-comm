package writer

import (
	"encoding/csv"
	"fmt"
	"github.com/YasiruR/didcomm-prober/scripts/tester/group"
	"log"
	"os"
	"strconv"
)

func Persist(test string, gc group.Config, results []float64) {
	switch test {
	case `join`:
		writeJoin(gc, results)
	}
}

func writeJoin(gc group.Config, results []float64) {
	const path = `results/join.csv`
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
		if err = w.Write([]string{`name`, `initial_size`, `mode`, `consistent_join`, `causally_ordered`, `zmq_buf_ms`, `latency_ms`}); err != nil {
			log.Fatalln(`writing join headers failed -`, err)
		}
	}

	for _, r := range results {
		if err = w.Write([]string{
			gc.Topic,
			strconv.FormatInt(gc.InitSize, 10),
			gc.Mode,
			fmt.Sprintf(`%t`, gc.ConsistntJoin),
			fmt.Sprintf(`%t`, gc.Ordered),
			strconv.FormatInt(gc.ZmqBuf, 10),
			fmt.Sprintf(`%f`, r),
		}); err != nil {
			log.Fatalln(`writing join results failed -`, err)
		}
	}

	w.Flush()
}
