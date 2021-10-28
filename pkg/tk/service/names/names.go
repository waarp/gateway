// Package names defines the names of the core gateway services.
package names

const (
	// DatabaseServiceName is the name of the gatewayd database service.
	DatabaseServiceName = "Database"

	// AdminServiceName is the name of the administration interface service.
	AdminServiceName = "Admin"

	// ControllerServiceName is the name of the controller service.
	ControllerServiceName = "Controller"
)

// IsReservedServiceName returns whether the given service name is already a
// reserved name. Reserved names cannot be used as service names.
func IsReservedServiceName(name string) bool {
	return name == DatabaseServiceName || name == AdminServiceName ||
		name == ControllerServiceName
}
