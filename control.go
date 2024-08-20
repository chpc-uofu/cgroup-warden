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

type controlProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type controlRequest struct {
	Unit     *string         `json:"unit"`
	Username *string         `json:"username"`
	Property controlProperty `json:"property"`
	Runtime  bool            `json:"runtime"`
}

type controlResponse struct {
	Unit     string          `json:"unit"`
	Username string          `json:"username"`
	Property controlProperty `json:"property"`
}

func controlHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var request controlRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			slog.Warn("unable to decode json request", "err", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		unit, username, err := resolveUser(request)
		if err != nil {
			slog.Warn("unable to resolve user", "err", err.Error(), "request", request)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		variant := dbus.MakeVariant(request.Property.Value)
		property := systemd.Property{Name: request.Property.Name, Value: variant}

		sysconn, err := newSystemdConn()
		if err != nil {
			slog.Warn("unable to connect to systemd", "err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer sysconn.conn.Close()

		err = sysconn.conn.SetUnitPropertiesContext(sysconn.ctx, unit, request.Runtime, property)
		if err != nil {
			slog.Warn("unable to set property", "err", err.Error(), "property", property, "unit", unit)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		response := controlResponse{Unit: unit, Username: username, Property: request.Property}
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			slog.Error("unable to send encode response", "error", err.Error())
		}
	}
}

func resolveUser(request controlRequest) (string, string, error) {

	var unit string
	var username string
	var err error

	if request.Unit != nil {
		username, err = getUsername(*request.Unit)
		return *request.Unit, username, err
	}

	if request.Username != nil {
		unit, err := getUnit(*request.Username)
		return unit, *request.Username, err
	}

	return unit, username, errors.New("must provide unit or username")
}

func getUsername(unit string) (string, error) {
	re := regexp.MustCompile(`user-(\d+)\.slice`)
	match := re.FindStringSubmatch(unit)
	if len(match) != 2 {
		return "", errors.New("invalid unit string")
	}
	usr, err := user.LookupId(match[1])
	return usr.Username, err
}

func getUnit(username string) (string, error) {
	usr, err := user.Lookup(username)
	if err != nil {
		return "", err
	}
	unit := fmt.Sprintf("user-%v.slice", usr.Uid)
	return unit, err
}
