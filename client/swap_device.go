package main

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// SwapDevice is the action to broadcast to the server, requesting for actions to be performed.
type SwapDevice struct {
	Device    string `json:"device"`
	Direction string `json:"direction"`
}

// Unmarshal reads in SwapDevice commands that may have been sent from other clients.
func (sd *SwapDevice) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, sd)
	if err != nil {
		return err
	}

	fields := reflect.ValueOf(sd).Elem()
	for i := 0; i < fields.NumField(); i++ {

		dsTags := fields.Type().Field(i).Tag.Get("kvm")
		if strings.Contains(dsTags, "required") && fields.Field(i).IsZero() {
			return errors.New("required field is missing")
		}

	}
	return nil
}