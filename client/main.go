package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/timgws/kvm-switch/client/icon"

	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
)

const (
	DisplayRefreshRate = 60
	EnableDebugMode = true
)

var messages chan string
var outgoingChanges = make(chan string)

var addr = flag.String("addr", "127.0.0.1:8787", "http service address")
var deviceName = flag.String("name", "home-computer", "the name of the device in the layout")

var cQuit <- chan bool

var maxX, maxY int
var lastX, lastY = 0, 0
var switched = false
var switching = false

func main() {
	flag.Parse()
	systray.Run(onReady, onExit)
	go PollMousePosition()

	go func() {
		for {
			select {
			case quit := <-cQuit:
				if quit {
					os.Exit(1)
				}
			}
		}
	}()
}

func PollMousePosition() {
	go doEvery(time.Second/DisplayRefreshRate, pollMousePosition)
	maxX, maxY = robotgo.GetScreenSize()
	fmt.Println("[Glide] get screen size: ", maxX, maxY)
}

func deviceHasAtLeastOneDisplay() bool {
	numDisplays := robotgo.DisplaysNum()
	// macOS will segfault when finding the mouse position if there are no displays attached.
	// TODO: If this does not crash on Windows, maybe only run on macOS
	if numDisplays == 0 {
		// TODO: Fix me, we should only show this once every $X minutes. This will flood the console.
		debugLog("[WARNING]: SKIPPING GET PIXEL/MOUSE POS BECAUSE THERE ARE NO SCREENS ATTACHED.")
		return false
	}

	return true
}

func pollMousePosition(t time.Time) {
	if !deviceHasAtLeastOneDisplay() {
		return
	}

	x, y := robotgo.GetMousePos()

	if !(x != lastX || y != lastY) {
		return
	}

	mouseHasMoved(x, y)
}

func mouseHasMoved(x int, y int) {
	if x == 0 && lastX > 0 && switched == false {
		switched = true
		switchInput("left")
	} else if x == maxX-1 && lastX < maxX-2 && switched == false {
		switched = true
		switchInput("right")
	} else if y == 0 && lastY > 0 && switched == false {
		switched = true
		switchInput("top")
	} else if y == maxY-1 && lastY < maxY-2 && switched == false {
		switched = true
		switchInput("bottom")
	}

	/*
	color := robotgo.GetPixelColor(100, 200)
	log.Printf("(mouse pos: %d x %d) pixel color @ 100x200: #%s", x, y, color)
	*/

	if EnableDebugMode {
		debugLog("[kvm-client] mouse pos (current: %d x %d, previous: %d x %d, max: %d x %d)", x, y, lastX, lastY, maxX, maxY)
	}

	lastX = x
	lastY = y
}

func switchInput(direction string) {
	if switching == false {
		debugLog("[Fence]: 📺 Mouse has moved to the %s desktop.", direction)
		switching = true
	}

	outgoingChanges <- makeSwap(SwapDevice{
		Device:    *deviceName,
		Direction: direction,
	})
}

func makeSwap(swap SwapDevice) string {
	j, _ := json.Marshal(swap)
	debugLog("--> Sending to kvm-server %s", j)
	return string(j)
}

func onReady() {
	log.Printf("[GUI] The application has booted")

	go connectWebsocket()
	go PollMousePosition()
	//go processInput()

	systray.SetIcon(icon.Data)
	systray.SetTitle("Fence")
	systray.SetTooltip("Fence: Move to corner, swap your input")
	mRefresh := systray.AddMenuItem("Refresh", "Refresh the state")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	// Sets the icon of a menu item. Only available on Mac and Windows.
	mQuit.SetIcon(icon.Data)
	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
	go func() {
		<-mRefresh.ClickedCh
		systray.SetTitle("Refreshing...")
		fmt.Println("Refreshing..")
	}()
}

// onExit runs after the GUI has asked to exit.
// TODO: clean up some sockets here.
func onExit() {
	// clean up here
}

// debugLog will output something only if EnableDebugMode is true.
func debugLog(msg string, v ...interface{}) {
	if EnableDebugMode {
		log.Printf(msg, v...)
	}
}
