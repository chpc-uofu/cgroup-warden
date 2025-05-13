package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/chpc-uofu/cgroup-warden/hierarchy"
	"github.com/containerd/cgroups/v3"
	systemd "github.com/coreos/go-systemd/v22/dbus"
	dbus "github.com/godbus/dbus/v5"
)

// properties that can be modified at runtime
var (
	CPUAccounting      = "CPUAccounting"
	CPUQuotaPerSecUSec = "CPUQuotaPerSecUSec"
	MemoryAccounting   = "MemoryAccounting"
	MemoryHigh         = "MemoryHigh"
	MemoryMax          = "MemoryMax"
	MemorySwapMax      = "MemorySwapMax"
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

func ControlHandler(cgroupRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		var request controlRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			slog.Warn("unable to decode json request", "err", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if request.Property.Name == MemorySwapMax && cgroups.Mode() == cgroups.Legacy {
			_, err = setCGroupMemorySwapLegacy(request, cgroupRoot)
		} else {
			err = setSystemdProperty(request)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "success")
	}
}

func setCGroupMemorySwapLegacy(request controlRequest, cgroupRoot string) (int64, error) {
	val, ok := request.Property.Value.(float64)
	if !ok {
		return -1, errors.New("invalid type for property, expected float64")
	}
	value := int64(val)
	h := hierarchy.NewHierarchy(cgroupRoot)
	return h.SetMemorySwap(request.Unit, value)
}

func setSystemdProperty(request controlRequest) error {
	property, err := transform(request.Property)
	if err != nil {
		slog.Warn("unable to create systemd property", "err", err.Error())
		return err
	}

	ctx := context.Background()
	conn, err := systemd.NewSystemConnectionContext(ctx)
	if err != nil {
		slog.Warn("unable to connect to systemd", "err", err.Error())
		return err
	}
	defer conn.Close()

	err = conn.SetUnitPropertiesContext(ctx, request.Unit, request.Runtime, property)
	if err != nil {
		slog.Warn("unable to set property", "err", err.Error(), "property", property, "unit", request.Unit)
		return err
	}
	return nil
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

	case MemorySwapMax:
		val, ok := controlProp.Value.(float64)
		if !ok {
			return property, errors.New("invalid type for property, expected float64")
		}

		// -1 not accepted as the 'unset' value, but rather 'max'
		if val == -1 {
			property.Value = dbus.MakeVariant("max")
		} else {

			property.Value = dbus.MakeVariant(uint64(val))
		}
		property.Value = dbus.MakeVariant(uint64(val))

	case CPUQuotaPerSecUSec, MemoryMax, MemoryHigh, MemoryMin, MemoryLow:
		val, ok := controlProp.Value.(float64) // json type
		if !ok {
			return property, errors.New("invalid type for property, expected float64")
		}
		property.Value = dbus.MakeVariant(uint64(val))

	default:
		msg := fmt.Sprintf("property not supported: %v", controlProp.Name)
		return property, errors.New(msg)
	}

	return property, nil

}
