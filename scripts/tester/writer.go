package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
)

func persist(test string, gc groupCfg, results []float64) {
	switch test {
	case `join`:
		writeJoin(gc, results)
	}
}

func writeJoin(gc groupCfg, results []float64) {
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
			gc.name,
			strconv.FormatInt(gc.initSize, 10),
			gc.mode,
			fmt.Sprintf(`%t`, gc.consistntJoin),
			fmt.Sprintf(`%t`, gc.ordered),
			strconv.FormatInt(gc.zmqBuf, 10),
			fmt.Sprintf(`%f`, r),
		}); err != nil {
			log.Fatalln(`writing join results failed -`, err)
		}
	}

	w.Flush()
}
