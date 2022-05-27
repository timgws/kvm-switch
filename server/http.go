package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func serveLayout(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	w.Header().Add("Content-Type", "application/json")
	_, err := w.Write(JSONLayout)
	if err != nil {
		fmt.Println(err)
	}
}

func serveDriverStatus(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	w.Header().Add("Content-Type", "application/json")

	drivers, _ := json.Marshal(Drivers)

	_, err := w.Write(drivers)
	if err != nil {
		fmt.Println(err)
	}
}

func serveRefreshStatus(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	for _, driver := range Drivers.Drivers {
		driver.GetStatus()
	}
}

func serveSwap(w http.ResponseWriter, r *http.Request) {
	actions, _ := TheLayout.FindActions("home-computer", "left")
	TheLayout.Effect(actions)

	time.Sleep(500 * time.Millisecond)
}