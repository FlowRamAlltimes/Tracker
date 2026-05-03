package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

type Stats struct {
	CPU      string `json:"cpu"`
	MemTotal int    `json:"totalram"`
	MemFree  int    `json:"freeram"`
	Usage    int    `json:"filledram"`
}

var (
	Db    *sql.DB
	Query string = `
	CREATE TABLE IF NOT EXISTS scans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME,
    ip TEXT,
    cpu_model TEXT,
    mem_total INTEGER,
    mem_free INTEGER,
    usage_percent INTEGER
);`
	PreparedQuery string = `
	INSERT INTO scans(timestamp ,ip, cpu_model, mem_total, mem_free, usage_percent) VALUES(?, ?, ?, ?, ?, ?)`
)

func main() {
	var err error
	Db, err = sql.Open("sqlite", "results.db")
	if err != nil {
		log.Println(err)
		return
	}
	defer Db.Close()

	_, err = Db.Exec(Query)
	if err != nil {
		log.Println(err)
		return
	}

	mux := http.NewServeMux()
	h := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		IdleTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
	log.Println("Server runs: http://localhost:8080")

	mux.HandleFunc("/report", reportFunc)
	mux.HandleFunc("/", mainpage)
	h.ListenAndServe()
}

func mainpage(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Hello from Golang!")
}

func reportFunc(w http.ResponseWriter, req *http.Request) {
	doPrepare, err := Db.Prepare(PreparedQuery)
	if err != nil {
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	ip := req.RemoteAddr

	key := req.Header.Get("X-API-KEY")
	if key != "My-API-Key" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	if req.Method != http.MethodPost {
		http.Error(w, "Method isnt allowed", http.StatusMethodNotAllowed)
		return
	}

	var s Stats

	err = json.NewDecoder(req.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = doPrepare.Exec(time.Now().Local(), ip, s.CPU, s.MemTotal, s.MemFree, s.Usage)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("[%s] CPU: %s | Total RAM: %d(%d%%) | Free RAM: %d\n", ip, s.CPU, s.MemTotal, s.Usage, s.MemFree)
}
