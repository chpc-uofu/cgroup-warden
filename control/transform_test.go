package control

import "testing"

func TestTransformNegativeCPUQuotaIsInfinity(t *testing.T) {
	prop, err := transform(controlProperty{Name: CPUQuotaPerSecUSec, Value: float64(-1)})
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	if got := prop.Value.Value(); got != ^uint64(0) {
		t.Errorf("value = %v, want %v", got, ^uint64(0))
	}
}
