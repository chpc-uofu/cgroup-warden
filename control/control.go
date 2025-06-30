package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/chpc-uofu/cgroup-warden/hierarchy"
	//"github.com/containerd/cgroups/v3"
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

type controlResponse struct {
	Unit     string          `json:"unit"`
	Property controlProperty `json:"property"`
	Error    string          `json:"error,omitempty"`
	Warning  string          `json:"warning,omitempty"`
}

func ControlHandler(cgroupRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		var err error
		var response controlResponse
		status := http.StatusOK

		defer func() {
			if err != nil {
				response.Error = err.Error()
			}

			w.WriteHeader(status)
			json.NewEncoder(w).Encode(response)
		}()

		var request controlRequest
		err = json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			slog.Warn("unable to decode json request", "err", err.Error())
			status = http.StatusBadRequest
			return
		}

		response.Unit = request.Unit
		response.Property = request.Property

		slog.Debug("Decoded request", "unit", request.Unit, "property", request.Property.Name, "value", request.Property.Value)
		
		var newLimit int64
		var fallback bool = false

		if request.Property.Name == MemorySwapMax || request.Property.Name == MemoryMax {
			newLimit, fallback, err = setCGroupMemoryLimits(request, cgroupRoot)

			if newLimit == hierarchy.MaxCGroupMemoryLimit{
				response.Property.Value = -1
			} else {
				response.Property.Value = newLimit
			}

			if fallback {
				response.Warning = fmt.Sprintf("unable to clamp memory limit down, defaulted to current usage %d", newLimit)
			}
		} else {
			err = setSystemdProperty(request)
		}

		if err != nil {
			status = http.StatusBadRequest
			return
		}
	}
}

func setCGroupMemoryLimits(request controlRequest, cgroupRoot string) (int64, bool, error) {
	val, ok := request.Property.Value.(float64)
	if !ok {
		return -1, false, errors.New("invalid type for property, expected float64")
	}

	value := int64(val)
	if value == -1 {
		value = hierarchy.MaxCGroupMemoryLimit
	}

	h := hierarchy.NewHierarchy(cgroupRoot)
	newLimit, err := h.SetMemoryLimits(request.Unit, value)

	fallback := (newLimit != value && newLimit != -1)

	return newLimit, fallback, err
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
	case CPUQuotaPerSecUSec, MemoryMax, MemoryHigh, MemoryMin, MemoryLow, MemorySwapMax:
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
