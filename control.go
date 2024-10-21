package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	systemd "github.com/coreos/go-systemd/v22/dbus"
	dbus "github.com/godbus/dbus/v5"
)

// properties that can be modified at runtime,
// see
var (
	CPUAccounting      = "CPUAccounting"
	CPUQuotaPerSecUSec = "CPUQuotaPerSecUSec"
	MemoryAccounting   = "MemoryAccounting"
	MemoryHigh         = "MemoryHigh"
	MemoryMax          = "MemoryMax"
	MemoryLow          = "MemoryLow"
	MemoryMin          = "MemoryMin"
)

type controlProperty struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type controlRequest struct {
	Unit     string          `json:"unit"`
	Property controlProperty `json:"property"`
	Runtime  bool            `json:"runtime"`
}

var ControlHandler = http.HandlerFunc(controlHandler)

func controlHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var request controlRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		slog.Warn("unable to decode json request", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	property, err := transform(request.Property)
	if err != nil {
		slog.Warn("unable to create systemd property", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	conn, err := systemd.NewSystemConnectionContext(ctx)
	if err != nil {
		slog.Warn("unable to connect to systemd", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	err = conn.SetUnitPropertiesContext(ctx, request.Unit, request.Runtime, property)
	if err != nil {
		slog.Warn("unable to set property", "err", err.Error(), "property", property, "unit", request.Unit)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "success")
}

func transform(controlProp controlProperty) (systemd.Property, error) {
	var property systemd.Property
	property.Name = controlProp.Name
	switch controlProp.Name {
	case CPUAccounting, MemoryAccounting:
		val, ok := controlProp.Value.(bool)
		if !ok {
			return property, errors.New("invalid type for property, expected bool")
		}
		property.Value = dbus.MakeVariant(val)

	case CPUQuotaPerSecUSec, MemoryMax, MemoryHigh, MemoryMin, MemoryLow:
		val, ok := controlProp.Value.(float64) // json type
		if !ok {
			return property, errors.New("invalid type for property, expected float64")
		}
		property.Value = dbus.MakeVariant(uint(val))

	default:
		msg := fmt.Sprintf("property not supported: %v", controlProp.Name)
		return property, errors.New(msg)
	}

	return property, nil

}
