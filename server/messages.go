package main

type Direction int64
const (
	Left Direction = iota
	Right
	Top
	Bottom
)

// IncomingAction is what is received by a client, connecting to a server.
// Server is to determine what actions (if any) are to be performed after receiving an action/command.
// { "action_name": "switch_direction", "action_direction": Left }
type IncomingAction struct {
	ActionName string
	ActionDirection Direction `json:"optional"`
}

// BroadcastAction is what will be sent to all connected clients when an operation has been performed by the server
// { "action_name": "active_computer", "value": "pc1" }
type BroadcastAction struct {
	ActionName string
}