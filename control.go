package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/user"
	"regexp"
	"strconv"

	systemd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	dbus "github.com/godbus/dbus/v5"
)

type systemdConn struct {
	conn *systemd.Conn
	ctx  context.Context
}

func newSystemdConn() (systemdConn, error) {
	ctx := context.Background()
	conn, err := systemd.NewSystemConnectionContext(ctx)
	return systemdConn{conn, ctx}, err
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type controlRequest struct {
	Unit     *string  `json:"unit"`
	Username *string  `json:"username"`
	Property property `json:"property"`
	Runtime  bool     `json:"runtime"`
}

type controlResponse struct {
	Message  string `json:"message"`
	Unit     string `json:"unit,omitempty"`
	Username string `json:"username,omitempty"`
}

func controlHandler(logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var request controlRequest
		var response controlResponse

		err = json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response.Unit, response.Username, err = resolveUser(request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		property, err := constructProperty(request.Property)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sysconn, err := newSystemdConn()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer sysconn.conn.Close()

		err = sysconn.conn.SetUnitPropertiesContext(sysconn.ctx, response.Unit, request.Runtime, property)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			level.Error(logger).Log("msg", "error sending response", "err", err)
		}
	}
}

func constructProperty(candidate property) (systemd.Property, error) {
	var value any
	var err error
	var property systemd.Property
	switch candidate.Name {
	case "MemoryMax":
		value, err = strconv.ParseFloat(candidate.Value, 64)
	case "CPUQuotaPerSecUSec":
		value, err = strconv.ParseFloat(candidate.Value, 64)
	case "MemoryAccounting":
		value, err = strconv.ParseBool(candidate.Value)
	case "CPUAccounting":
		value, err = strconv.ParseBool(candidate.Value)
	default:
		value, err = nil, fmt.Errorf("%v is not a valid property", candidate.Name)
	}

	if err != nil {
		return property, err
	}

	property.Value = dbus.MakeVariant(value)
	property.Name = candidate.Name
	return property, err
}

func resolveUser(request controlRequest) (username string, unit string, err error) {
	if request.Unit != nil && request.Username != nil {
		username, err = resolveUsername(*request.Unit)
		if err != nil {
			return
		}
		unit, err = resolveUnit(*request.Unit)
		if err != nil {
			return
		}
		if username != *request.Username || unit != *request.Unit {
			err = errors.New("unit and username do not match")
			return
		}
	}
	if request.Unit != nil {
		username, err = resolveUsername(*request.Unit)
		unit = *request.Unit
		return
	}
	if request.Username != nil {
		unit, err = resolveUnit(*request.Username)
		username = *request.Username
		return
	}
	err = errors.New("must provide unit or username")
	return
}

func resolveUsername(unit string) (username string, err error) {
	re := regexp.MustCompile(`user-(\d+)\.slice`)
	match := re.FindStringSubmatch(unit)
	var usr *user.User
	if len(match) != 2 {
		err = errors.New("invalid unit")
		return
	}
	usr, err = user.LookupId(match[1])
	if err != nil {
		return
	}
	return usr.Username, nil
}

func resolveUnit(username string) (unit string, err error) {
	var usr *user.User
	usr, err = user.Lookup(username)
	if err != nil {
		return
	}
	unit = fmt.Sprintf("user-%v.slice", usr.Uid)
	return
}
