package models

import "testing"

func TestIsValidRole(t *testing.T) {
	valid := []Role{RoleAdmin, RoleVendor, RoleCustomer, RoleMechanic}
	for _, r := range valid {
		if !IsValidRole(r) {
			t.Fatalf("expected valid role %q", r)
		}
	}
	if IsValidRole(Role("INVALID")) {
		t.Fatal("expected invalid role to fail")
	}
}

func TestValidRolesIncludesMechanic(t *testing.T) {
	roles := ValidRoles()
	found := false
	for _, r := range roles {
		if r == RoleMechanic {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("ValidRoles should include MECHANIC")
	}
}

func TestIsValidMechanicProfileStatus(t *testing.T) {
	for _, s := range []MechanicProfileStatus{
		MechanicStatusPending,
		MechanicStatusVerified,
		MechanicStatusSuspended,
		MechanicStatusRejected,
	} {
		if !IsValidMechanicProfileStatus(s) {
			t.Fatalf("expected valid status %q", s)
		}
	}
	if IsValidMechanicProfileStatus(MechanicProfileStatus("unknown")) {
		t.Fatal("expected invalid status to fail")
	}
}
