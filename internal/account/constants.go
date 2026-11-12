package account

const (
	// RoleAdmin is the persisted account role.
	RoleAdmin = "admin"
	// RoleUser is the persisted account role.
	RoleUser = "user"
	// StatusActive is the persisted account status.
	StatusActive = "active"
	// StatusInactive is the persisted account status.
	StatusInactive = "inactive"
	// StatusInvited is the persisted account status.
	StatusInvited = "invited"

	passwordIterations = 2
	passwordKeyBytes   = 32
	passwordSaltBytes  = 16

	retryBaseSeconds = 15
)

const (
	passwordMemoryKiB = 19 * 1024
)
