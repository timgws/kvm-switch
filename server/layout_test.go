package main

import (
	"encoding/json"
	"github.com/timgws/kvm-switch/server/drivers/blustream"
	"github.com/timgws/kvm-switch/server/drivers/startech_kvm"
	"log"
	"testing"
)

func TestLayoutDirection(t *testing.T) {
	layout := BuildLayout()
	actions, err := layout.FindActions("home-computer", "left")

	if len(*actions) > 0 {
		t.Logf("Found %d actions...", len(*actions))
	}

	if err != nil {
		t.Fatalf("There was an error: %s", err)
	}
}

func TestLayoutWithDriver(t *testing.T) {
	Drivers.Drivers = append(Drivers.Drivers, blustream.NewInstance())
	Drivers.Drivers = append(Drivers.Drivers, startech_kvm.NewInstance())

	layout := BuildLayout()

	actions, _ := layout.FindActions("home-computer", "left")
	k, _ := json.Marshal(actions)
	log.Printf("%s", k)

	actions, err := layout.FindActions("home-computer", "right")
	k, _ = json.Marshal(actions)
	log.Printf("%s %s", k, err)

	//layout.Effect(actions)
}