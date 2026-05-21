package models

// ValidRoles returns all assignable user roles.
func ValidRoles() []Role {
	return []Role{RoleAdmin, RoleVendor, RoleCustomer, RoleMechanic}
}

// IsValidRole reports whether r is an allowed role value.
func IsValidRole(r Role) bool {
	for _, valid := range ValidRoles() {
		if r == valid {
			return true
		}
	}
	return false
}
