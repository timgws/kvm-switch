package main

import (
	"fmt"
	"github.com/jpillora/backoff"
	"golang.org/x/net/websocket"
	"time"
)

var (
	websocketConnected = false
	websocketError = make (chan error)
)

func connectWebsocket() {
	b := &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: false,
	}

	debugLog("Starting Client")

	for {
		err := initWebsocketClient()
		if err != nil {
			d := b.Duration()
			debugLog("%s, reconnecting in %s", err, d)
			time.Sleep(d)
			continue
		} else {
			b.Reset()
		}


		select {
		case wserr := <-websocketError:
			if websocketConnected {
				debugLog("[Websocket]: error: %s... will reconnect", wserr)
				websocketConnected = false
			}
		}
	}
}

func initWebsocketClient() error {
	websocketServer := fmt.Sprintf("ws://%s/ws", *addr)
	ws, err := websocket.Dial(websocketServer, "", fmt.Sprintf("http://%s/", *addr))
	if err != nil {
		debugLog("Dial failed: %s\n", err.Error())
		return err
	}

	incomingMessages := make(chan string)
	go readClientMessages(ws, incomingMessages)
	i := 0

	websocketConnected = true
	debugLog("Connected to server: %s", websocketServer)
	for {
		select {
		case send := <-outgoingChanges:
			if websocketConnected {
				err = websocket.Message.Send(ws, send)
			}
		case <-time.After(time.Duration(2e9)):
			i++
		case message, ok := <-incomingMessages:
			if !ok {
				incomingMessages = nil
				outgoingChanges = nil
			}
			fmt.Println(`[websocket] Message Received:`, message)

			var sd SwapDevice
			if err := sd.Unmarshal([]byte(message)); err == nil {
				switched = false
				switching = false
			}
		}

		if incomingMessages == nil && outgoingChanges == nil {
			break
		}
	}

	return nil
}

func readClientMessages(ws *websocket.Conn, incomingMessages chan string) {
	for {
		var message string
		err := websocket.Message.Receive(ws, &message)
		if err != nil {
			debugLog("Websocket Error: %s\n", err.Error())

			//close(outgoingChanges)
			close(incomingMessages)
			websocketError <- err
			return
		}

		if message != "" {
			incomingMessages <- message
		}
	}
}
