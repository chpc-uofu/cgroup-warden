package control

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateUnit(t *testing.T) {
	valid := []string{
		"user-0.slice",
		"user-1000.slice",
		"user-123456.slice",
	}
	for _, unit := range valid {
		if err := validateUnit(unit); err != nil {
			t.Errorf("validateUnit(%q) = %v, want nil", unit, err)
		}
	}

	invalid := []string{
		"",
		"-.slice",
		"user.slice",
		"system.slice",
		"init.scope",
		"sshd.service",
		"user-1000.scope",
		"user-abc.slice",
		"../system.slice/sshd.service",
		"user-1000.slice/../../system.slice",
		"user-1000.slice/",
		"user-1000.slice ",
		"user-1000.slice\n",
	}
	for _, unit := range invalid {
		if err := validateUnit(unit); err == nil {
			t.Errorf("validateUnit(%q) = nil, want error", unit)
		}
	}
}

func TestControlHandlerRejectsNonUserUnits(t *testing.T) {
	handler := ControlHandler("/user.slice")

	for _, unit := range []string{
		"",
		"system.slice",
		"init.scope",
		"sshd.service",
		"user.slice",
		"../system.slice/sshd.service",
	} {
		body, err := json.Marshal(controlRequest{
			Unit: unit,
			Property: controlProperty{
				Name: "CPUQuotaPerSecUSec",
				// Non-numeric: if the unit check regresses, transform() still
				// rejects the request before systemd is contacted.
				Value: "x",
			},
		})
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/control", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("unit %q: status = %d, want %d", unit, rec.Code, http.StatusBadRequest)
		}

		var response controlResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Errorf("unit %q: response is not JSON: %v", unit, err)
			continue
		}
		if want := validateUnit(unit).Error(); response.Error != want {
			t.Errorf("unit %q: error = %q, want %q", unit, response.Error, want)
		}
	}
}
