package main

import (
	"fmt"
	"github.com/timgws/kvm-switch/server/drivers"
	"strings"
)

// Layout defines where the machines are, and what the actions that will be performed when
// the mouse is moved around different areas.
type Layout struct {
	Computers []Computer
}

// Computer is a computer (or device) that will be swapped on the matrix.
type Computer struct {
	Name string
	Directions Directions
}

// Directions for computers were actions will be performed when moving the mouse between different areas.
type Directions struct {
	Left []Action
	Right []Action
	Top []Action
	Bottom []Action
}

// Action defines an individual _thing_ that will happen after an action is performed.
type Action struct {
	DriverName string
	PerformAction string
}

// BuildLayout should build a layout that is usable by the clients.
func BuildLayout() *Layout {
	return &Layout{
		Computers: []Computer{{
			Name: "work-computer",
			Directions: Directions{
				Right: []Action{{
					DriverName: "kvm",
					PerformAction: "1",
				}},
			},
		}, {
			Name: "home-computer",
			Directions: Directions{
				Left: []Action{{
					DriverName: "kvm",
					PerformAction: "2",
				}},
				Right: []Action{{
					DriverName: "matrix",
					PerformAction: "01-03",
				}, {
					DriverName: "matrix",
					PerformAction: "02-04",
				}, {
					DriverName: "kvm",
					PerformAction: "4",
				}},
			},
		}, {
			Name: "streaming-computer",
			Directions: Directions{
				Left: []Action{{
					DriverName: "matrix",
					PerformAction: "01-01",
				}, {
					DriverName: "matrix",
					PerformAction: "02-02",
				}, {
					DriverName: "kvm",
					PerformAction: "2",
				}},
			},
		}},
	}
}

// FindActions with a given computer/device, will find the actions that will be performed.
func (l *Layout) FindActions(name string, direction string) (*[]Action, error) {
	var ourDevice *Computer
	for i, devices := range l.Computers {
		if devices.Name == name {
			ourDevice = &l.Computers[i]
		}
	}

	if ourDevice == nil {
		return nil, fmt.Errorf("device [%s] was not found", name)
	}

	var actions *[]Action
	switch direction {
	case "left":
		actions = &ourDevice.Directions.Left
	case "right":
		actions = &ourDevice.Directions.Right
	case "top":
		actions = &ourDevice.Directions.Top
	case "bottom":
		actions = &ourDevice.Directions.Bottom
	}

	if len(*actions) > 0 {
		return actions, nil
	}

	return nil, nil
}

// Effect tells the drivers to perform the actions for a given device in the matrix.
func (l *Layout) Effect (actions *[]Action) {
	if actions == nil || len(*actions) < 1 {
		return
	}
	for _, item := range *actions {
		driver := findDriver(item.DriverName)
		action := item.PerformAction

		switch d := driver.(type) {
		case drivers.OutputSingle:
			d.SetOutput(action)
		case drivers.OutputMatrix:
			if strings.Contains(action, "-") {
				inOut := strings.Split(action, "-")
				d.SetOutput(inOut[0], inOut[1])
			}
		}
	}
}