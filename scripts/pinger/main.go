package main

import (
	"encoding/csv"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	pingPort = 9797
)

func main() {
	args := os.Args
	switch args[1] {
	case `client`:
		testPing()
	case `server`:
		server()
	}
}

func server() {
	r := mux.NewRouter()
	r.HandleFunc(`/ping`, handlePing).Methods(http.MethodGet)
	if err := http.ListenAndServe(":"+strconv.Itoa(pingPort), r); err != nil {
		log.Fatalln(fmt.Sprintf(`http server initialization failed - %v`, err))
	}
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`ok`))
}

func testPing() {
	labels, ips := read()
	for i, ip := range ips {
		fmt.Printf("# ping test to %s (%s)\n", labels[i], ip)
		var total int64
		for j := 0; j < 3; j++ {
			latency, err := ping(ip)
			if err != nil {
				fmt.Printf("	> attempt %d failed: %s\n", j, err)
				continue
			}
			total += latency
			fmt.Printf("	> attempt %d: %d ms\n", j, latency)
		}
		fmt.Printf("  average: %d ms\n\n", total/3)
	}
}

func read() (labels []string, ips []string) {
	f, err := os.Open(`remote.csv`)
	if err != nil {
		log.Fatalln(fmt.Sprintf(`opening file failed - %v`, err))
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatalln(`reading nodes failed -`, err)
	}

	for _, row := range records {
		labels = append(labels, row[0])
		ips = append(ips, row[1])
	}

	return
}

func ping(ip string) (int64, error) {
	start := time.Now()
	res, err := http.Get(`http://` + ip + `:` + strconv.Itoa(pingPort) + `/ping`)
	if err != nil {
		return 0, fmt.Errorf(`request failed - %v`, err)
	}

	latency := time.Since(start).Milliseconds()
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, fmt.Errorf(`reading response failed - %v`, err)
	}

	if string(data) != `ok` {
		return 0, fmt.Errorf(`server err - %v`, err)
	}

	return latency, nil
}
