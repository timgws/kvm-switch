package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/timgws/kvm-switch/server/drivers"
	"github.com/timgws/kvm-switch/server/drivers/blustream"
	"github.com/timgws/kvm-switch/server/drivers/startech_kvm"
)

const (
	maxBufferSize = 1024
	EnableDebugMode = true
)
var doneChan = make(chan error, 1)
var buffer = make([]byte, maxBufferSize)

var serverContext context.Context
var serverContextCtx context.Context
var serverContextCancel context.Context

var addr = flag.String("addr", ":8787", "http service address")

type allDrivers struct {
	Drivers []drivers.DriverInterface
}
var Drivers allDrivers
var TheLayout *Layout
var JSONLayout []byte


func main() {
	generateLayout()
	registerDrivers()
	startDrivers()

	flag.Parse()

	hub := newHub()
	go hub.run()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/layout", serveLayout)
	http.HandleFunc("/driverStatus", serveDriverStatus)
	http.HandleFunc("/refreshStatus", serveRefreshStatus)
	http.HandleFunc("/swap/:driver/:input/:output", serveSwap)
	http.HandleFunc("/swap", serveSwap)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Printf("Failed to start websocket: ListenAndServe: %s", err)
	}
}

// generateLayout should generate a layout from configuration.
// TODO: fix the should.
func generateLayout() {
	TheLayout = BuildLayout()
	if (EnableDebugMode) {
		JSONLayout, _ = json.Marshal(TheLayout)
		log.Printf("Layout JSON: %s", string(JSONLayout))
	}
}

// registerDrivers should go through config, and init the correct drivers required for the layout.
// TODO: fix the 'should'
func registerDrivers() {
	stkvm := startech_kvm.NewInstance()
	Drivers.Drivers = append(Drivers.Drivers, stkvm)
	bsmatrix := blustream.NewInstance()
	Drivers.Drivers = append(Drivers.Drivers, bsmatrix)
}

// startDrivers will start all registered drivers.
func startDrivers() {
	for _, driver := range Drivers.Drivers {
		starting := driver.Start()
		if EnableDebugMode {
			log.Printf("Started driver: %s (did it attempt to start? %b)", driver.DriverName(), starting)
		}
		if driver.LastError() != nil {
			log.Printf("[%s]: ERROR: %e", driver.DriverName(), driver.LastError())
		}
	}
}

// findDriver will return an instance of a driver with the given shortName that the driver was configured with.
func findDriver(shortName string) drivers.DriverInterface {
	for _, driver := range Drivers.Drivers {
		if driver.GetShortName() == shortName {
			return driver
		}

		switch d := driver.(type) {
		default:
		case drivers.Driver:
			if d.ShortName == shortName {
				return &d
			}
		}
	}
	return nil
}
