package main

import "testing"

func TestConstructProperty(t *testing.T) {

	testCases := []property{
		{Name: "CPUSecPerUSec", Value: "-1"},
		{Name: "MemoryAccounting", Value: "-1"},
		{Name: "MemoryMax", Value: "100000000"},
		{Name: "CPUAccounting", Value: "true"},
	}

	for _, tc := range testCases {
		_, err := constructProperty(tc)
		if err != nil {
			t.Fail()
		}
	}

}

func TestResolveUser(t *testing.T) {
	var prop property
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
