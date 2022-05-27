package main

import (
	"fmt"
	"golang.org/x/net/websocket"
	"os"
	"time"
)


func connectWebsocket() {
	initWebsocketClient()
}

func initWebsocketClient() {
	fmt.Println("Starting Client")
	ws, err := websocket.Dial(fmt.Sprintf("ws://%s/ws", *addr), "", fmt.Sprintf("http://%s/", *addr))
	if err != nil {
		fmt.Printf("Dial failed: %s\n", err.Error())
		os.Exit(1)
	}

	incomingMessages := make(chan string)
	go readClientMessages(ws, incomingMessages)
	i := 0

	for {
		select {
		case send := <-outgoingChanges:
			err = websocket.Message.Send(ws, send)
		case <-time.After(time.Duration(2e9)):
			i++
		case message := <-incomingMessages:
			fmt.Println(`[websocket] Message Received:`, message)

			var sd SwapDevice
			if err := sd.Unmarshal([]byte(message)); err == nil {
				switched = false
				switching = false
			}
		}
	}
}

func readClientMessages(ws *websocket.Conn, incomingMessages chan string) {
	for {
		var message string
		// err := websocket.JSON.Receive(ws, &message)
		err := websocket.Message.Receive(ws, &message)
		if err != nil {
			debugLog("Websocket Error: %s\n", err.Error())
			return
		}
		incomingMessages <- message
	}
}
