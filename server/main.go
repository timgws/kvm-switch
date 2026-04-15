package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/timgws/kvm-switch/server/config"
	"github.com/timgws/kvm-switch/server/drivers"
	"github.com/timgws/kvm-switch/server/drivers/blustream"
	"github.com/timgws/kvm-switch/server/drivers/startech_kvm"
	"github.com/timgws/kvm-switch/server/serialdiscovery"
)

var EnableDebugMode = true

var configPath = flag.String("config", "config.toml", "path to server config.toml")
var addr = flag.String("addr", "", "http service address (overrides config.server.addr)")

type allDrivers struct {
	Drivers []drivers.DriverInterface
}

var Drivers allDrivers
var TheLayout *Layout
var JSONLayout []byte


func main() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	listenAddr := cfg.Server.Addr
	if *addr != "" {
		listenAddr = *addr
	}

	setDebugMode(cfg)
	generateLayout(cfg)
	registerDrivers(cfg)
	startDrivers()

	hub := newHub()
	go hub.run()

	mux := http.NewServeMux()

	// Basic endpoints
	mux.HandleFunc("GET /", serveHome)
	mux.HandleFunc("GET /layout", serveLayout)
	mux.HandleFunc("GET /driverStatus", serveDriverStatus)
	mux.HandleFunc("POST /refreshStatus", serveRefreshStatus)
	mux.HandleFunc("POST /swap/{driver}/{input}/{output}", serveSwap)
	mux.HandleFunc("POST /action/{name}/{direction}", serveAction)

	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r, cfg)
	})

	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Printf("Failed to start server: %v", err)
	}
}

func setDebugMode(cfg *config.Config) {
	EnableDebugMode = cfg.Server.Debug

	startech_kvm.EnableDebugMode = cfg.Server.Debug
	blustream.EnableDebugMode = cfg.Server.Debug

	for _, part := range cfg.Server.DebugModules {
		shouldDebug := !strings.HasPrefix(part, "!")
		if shouldDebug {
			part = strings.TrimPrefix(part, "!")
		}

		switch part {
		case "server":
			EnableDebugMode = shouldDebug
		case "startech_kvm":
			startech_kvm.EnableDebugMode = shouldDebug
		case "blustream":
			blustream.EnableDebugMode = shouldDebug
		}
	}
}

// generateLayout generates a layout from configuration.
func generateLayout(cfg *config.Config) {
	TheLayout = buildLayoutFromConfig(cfg)
	JSONLayout, _ = json.Marshal(TheLayout)
	if EnableDebugMode {
		log.Printf("Layout JSON: %s", string(JSONLayout))
	}
}

func buildLayoutFromConfig(cfg *config.Config) *Layout {
	l := &Layout{}
	for _, c := range cfg.Layout.Computers {
		l.Computers = append(l.Computers, Computer{
			Name: c.Name,
			Directions: Directions{
				Left:   convertActions(c.Left),
				Right:  convertActions(c.Right),
				Top:    convertActions(c.Top),
				Bottom: convertActions(c.Bottom),
			},
		})
	}
	return l
}

func convertActions(in []config.ActionConfig) []Action {
	out := make([]Action, 0, len(in))
	for _, a := range in {
		out = append(out, Action{
			DriverName:    a.Driver,
			PerformAction: a.Action,
		})
	}
	return out
}

// registerDrivers goes through the provided config, and prepares drivers required for the layout.
func registerDrivers(cfg *config.Config) {
	Drivers.Drivers = nil

	for _, dcfg := range cfg.Drivers {
		switch dcfg.Type {
		case "startech_kvm":
			drv := startech_kvm.NewInstanceWithConfig(startech_kvm.StartechConfig{
				SerialDevice: dcfg.SerialDevice,
				SerialBaud:   dcfg.Baud,
				AutoDiscover: dcfg.AutoDiscover,
				Match: serialdiscovery.Match{
					VID: dcfg.USBVID, PID: dcfg.USBPID,
					SerialNumber:    dcfg.SerialNumber,
					ProductContains: dcfg.ProductContains,
				},
				ShortName: dcfg.ShortName,
				Name:      dcfg.Name,
			})
			Drivers.Drivers = append(Drivers.Drivers, drv)

		case "blustream":
			drv := blustream.NewInstanceWithConfig(blustream.BlustreamConfig{
				SerialDevice: dcfg.SerialDevice,
				SerialBaud:   dcfg.Baud,
				AutoDiscover: dcfg.AutoDiscover,
				Match: serialdiscovery.Match{
					VID: dcfg.USBVID, PID: dcfg.USBPID,
					SerialNumber:    dcfg.SerialNumber,
					ProductContains: dcfg.ProductContains,
				},
				ShortName: dcfg.ShortName,
				Name:      dcfg.Name,
			})
			Drivers.Drivers = append(Drivers.Drivers, drv)

		default:
			log.Fatalf("unknown driver type: %q", dcfg.Type)
		}
	}
}

// startDrivers will start all registered drivers.
func startDrivers() {
	for _, driver := range Drivers.Drivers {
		starting := driver.Start()
		if EnableDebugMode {
			log.Printf("Started driver: %s (attempted=%t)", driver.DriverName(), starting)
		}
		if driver.LastError() != nil {
			log.Printf("[%s]: ERROR: %v", driver.DriverName(), driver.LastError())
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
