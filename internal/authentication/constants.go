package authentication

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
	StatusInvited      = "invited"
	inviteLifetimeDays = 7
	maxMailAttempts    = 8
	passwordIterations = 2
	passwordKeyBytes   = 32
	passwordSaltBytes  = 16
	purposeInvite      = "invite"
	purposeReset       = "reset"
	resetTokenSeconds  = 3600
	retryBaseSeconds   = 15
	smtpConnectSeconds = 10
	smtpWriteSeconds   = 20
)

const (
	cookieParts           = 2
	workerIntervalSeconds = 2
)

const (
	passwordMemoryKiB = 19 * 1024
	secondsPerDay     = 86400
)
