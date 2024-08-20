package main

import (
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
}

func TestGetUnit(t *testing.T) {
	username := "root"
	unit, err := getUnit(username)
	if err != nil || unit != "user-0.slice" {
		t.Fail()
	}
}

func TestSetMemoryAccounting(t *testing.T) {
	tc := controlProperty{Name: "CPUQuotaPerSecUSec", Value: "infinity"}

	sysconn, err := newSystemdConn()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer sysconn.conn.Close()
	unit := "user-1000.slice"
	property, err := transform(tc)
	if err != nil {
		t.Fatal(err.Error())
	}
	err = sysconn.conn.SetUnitPropertiesContext(sysconn.ctx, unit, true, property)
	if err != nil {
		t.Fatal(err.Error())
	}
}
