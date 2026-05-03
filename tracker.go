package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type postStruct struct {
	CPU      string `json:"cpu"`
	TotalRam int    `json:"totalram"`
	FreeRam  int    `json:"freeram"`
	Loading  int    `json:"filledram"`
}

func main() {
	dur := flag.Int("durat", 10, "time duration when programm sends POST requests")

	flag.Parse()

	var (
		cpuName  string
		totalMem int
		freeMem  int
		loading  int
		wg       sync.WaitGroup
	)
	for {
		log.Printf("Checking...")
		wg.Add(1)
		go func() {
			defer wg.Done()
			cpuName = getCpuInfo()
		}()
		wg.Wait()

		wg.Add(1)
		go func() {
			defer wg.Done()
			totalMem, freeMem, loading = getMemTotal()
		}()
		wg.Wait()

		log.Printf("CPU: %s, Total RAM: %d (%d), Free RAM: %d", cpuName, totalMem, loading, freeMem)

		p := &postStruct{
			CPU:      cpuName,
			TotalRam: totalMem,
			Loading:  loading,
			FreeRam:  freeMem,
		}

		p.sendData()
		time.Sleep(time.Duration(*dur * int(time.Second)))
	}
}

func getCpuInfo() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return err.Error()
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			res := strings.SplitN(line, ":", 2)
			return strings.TrimSpace(res[1])
		}
	}
	return ""
}

func getMemTotal() (int, int, int) {
	var (
		total   int
		free    int
		loading int
	)
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal") {
			res := strings.Fields(line)
			num, _ := strconv.Atoi(res[1])
			total = num / 1024
		} else if strings.HasPrefix(line, "MemFree") {
			res := strings.Fields(line)
			num, _ := strconv.Atoi(res[1])
			free = num / 1024
		}
	}
	tmpVal := total - free
	loading = (tmpVal * 100) / total

	return total, free, loading
}

func (p *postStruct) sendData() {
	c := &http.Client{
		Timeout: 5 * time.Second,
	}
	stream, _ := json.Marshal(p)

	req, err := http.NewRequest("POST", "http://localhost:8080/report", bytes.NewBuffer(stream))
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	req.Header.Set("X-API-KEY", "My-API-Key")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)

	log.Printf("Server response: %v", resp.Status)
}
