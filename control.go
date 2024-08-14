// Copyright (C) 2024 Center for High Performance Computing <helpdesk@chpc.utah.edu>

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/user"
	"regexp"

	systemd "github.com/coreos/go-systemd/v22/dbus"
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

func controlHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var request controlRequest
		var response controlResponse

		err = json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			slog.Warn("unable to decode json request", "err", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response.Unit, response.Username, err = resolveUser(request)
		if err != nil {
			slog.Warn("unable to resolve user", "err", err.Error(), "request", request)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		property, err := constructProperty(request.Property)
		if err != nil {
			slog.Warn("unable to construct  property", "err", err.Error(), "request", request)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sysconn, err := newSystemdConn()
		if err != nil {
			slog.Warn("unable to connect to systemd", "err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer sysconn.conn.Close()

		err = sysconn.conn.SetUnitPropertiesContext(sysconn.ctx, response.Unit, request.Runtime, property)
		if err != nil {
			slog.Warn("unable to set property", "err", err.Error(), "property", property, "unit", response.Unit)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			slog.Error("unable to send encode response", "error", err.Error())
		}
	}
}

func constructProperty(candidate property) (systemd.Property, error) {
	var property systemd.Property

	value, err := dbus.ParseVariant(candidate.Value, dbus.Signature{})
	if err != nil {
		return property, err
	}

	property.Value = value
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
