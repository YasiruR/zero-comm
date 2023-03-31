package main

import (
	"encoding/csv"
	"fmt"
	"github.com/YasiruR/didcomm-prober/reqrep/mock"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	pingPort = 9797
)

func main() {
	testPing()
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
	var avgLatList []float64
	labels, ips, mockPorts := read()
	for i, ip := range ips {
		fmt.Printf("# ping test to %s (%s:%s)\n", labels[i], ip, mockPorts[i])
		var total int64
		for j := 0; j < 3; j++ {
			latency, err := ping(ip, mockPorts[i])
			if err != nil {
				fmt.Printf("	> attempt %d failed: %s\n", j, err)
				continue
			}
			total += latency
			fmt.Printf("	> attempt %d: %d ms\n", j, latency)
		}

		avg := float64(total) / 3.0
		fmt.Printf("  average: %f ms\n\n", avg)
		avgLatList = append(avgLatList, avg)
	}

	groupByZone(labels, avgLatList)
}

func read() (labels, ips, mockPorts []string) {
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
		mockPorts = append(mockPorts, row[4])
	}

	return
}

func ping(ip, mockPort string) (int64, error) {
	start := time.Now()
	_, err := http.Get(`http://` + ip + `:` + mockPort + mock.PingEndpoint)
	if err != nil {
		return 0, fmt.Errorf(`request failed - %v`, err)
	}

	latency := time.Since(start).Milliseconds()
	return latency, nil
}

func groupByZone(labels []string, latList []float64) {
	fmt.Println()
	tmpLatMap := make(map[string][]float64)
	for i, l := range labels {
		name := strings.Split(l, "-")[0]
		tmpLatMap[name] = append(tmpLatMap[name], latList[i])
	}

	for name, lats := range tmpLatMap {
		var total float64
		for _, lat := range lats {
			total += lat
		}
		fmt.Printf("%s time-zone: %f ms\n", name, total/float64(len(lats)))
	}
}
