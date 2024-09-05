package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveUser(t *testing.T) {
	var prop controlProperty
	unit := "user-0.slice"
	username := "root"
	testCases := []controlRequest{
		{Unit: &unit, Username: nil, Property: prop, Runtime: false},
		{Unit: nil, Username: &username, Property: prop, Runtime: false},
		{Unit: &unit, Username: &username, Property: prop, Runtime: false},
	}

	for _, tc := range testCases {
		slice, name, err := resolveUser(tc)
		if err != nil || slice != unit || name != username {
			t.Fail()
		}
	}

	badRequest := controlRequest{Unit: nil, Username: nil, Property: prop, Runtime: false}
	_, _, err := resolveUser(badRequest)
	if err == nil {
		t.Fail()
	}
}

func TestGetUsername(t *testing.T) {
	unit := "user-0.slice"
	username, err := getUsername(unit)
	if err != nil || username != "root" {
		t.Fail()
	}

	unit = "NotAUnit"
	_, err = getUsername(unit)
	if err == nil {
		t.Fail()
	}
}

func TestGetUnit(t *testing.T) {
	username := "root"
	unit, err := getUnit(username)
	if err != nil || unit != "user-0.slice" {
		t.Fail()
	}

	username = "NotAUser"
	_, err = getUnit(username)
	if err == nil {
		t.Fail()
	}
}

func TestTransform(t *testing.T) {
	testCases := []controlProperty{
		{Name: "MemoryMax", Value: "-1"},
		{Name: "CPUQuotaPerSecUSec", Value: "4294967295"},
		{Name: "CPUAccounting", Value: "true"},
		{Name: "MemoryAccounting", Value: "0"},
	}

	for _, tc := range testCases {
		_, err := transform(tc)
		if err != nil {
			t.Fail()
		}

	}

	bad_prop := controlProperty{Name: "NotAProp", Value: "Bogus"}
	_, err := transform(bad_prop)
	if err == nil {
		t.Fail()
	}
}

// this test requires priviledge
func TestControlHandler(t *testing.T) {
	mockControlRequest := map[string]interface{}{
		"unit": "user-1000.slice",
		"property": map[string]interface{}{
			"name":  "CPUAccounting",
			"value": "true",
		},
		"runtime": true,
	}
	payload, _ := json.Marshal(mockControlRequest)
	req := httptest.NewRequest("POST", "/control", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	ControlHandler(w, req)
	resp := w.Result()
	var cr controlResponse
	err := json.NewDecoder(resp.Body).Decode(&cr)
	if err != nil {
		t.Fail()
	}
	if resp.StatusCode != http.StatusOK {
		t.Fail()
	}
}
