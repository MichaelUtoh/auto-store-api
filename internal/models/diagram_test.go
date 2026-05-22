package models

import "testing"

func TestVehicleSystemBeforeCreateSetsID(t *testing.T) {
	v := &VehicleSystem{Code: "brakes", Name: "Brakes"}
	if err := v.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if v.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Fatal("expected ID to be set")
	}
}
