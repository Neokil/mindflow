package authentication

// SetupCommand provides the one-time credential, first administrator and SMTP settings.
type SetupCommand struct {
	Credential string   `json:"credential"`
	Email      string   `json:"email"`
	Password   string   `json:"password"`
	Settings   Settings `json:"settings"`
}

// SetupTestEmailCommand tests first-run SMTP settings before setup is committed.
type SetupTestEmailCommand struct {
	Credential string   `json:"credential"`
	Email      string   `json:"email"`
	Settings   Settings `json:"settings"`
}

// LoginCommand contains the email and password supplied at sign-in.
type LoginCommand struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// InviteCommand requests an invited account with a role.
type InviteCommand struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// ForgotCommand requests a password reset without disclosing account existence.
type ForgotCommand struct {
	Email string `json:"email"`
}

// RedeemCommand sets a password using an invitation or reset token.
type RedeemCommand struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// UpdateUserCommand requests a role or activation-status change.
type UpdateUserCommand struct {
	Role   string `json:"role"`
	Status string `json:"status"`
}

// DeleteUserCommand confirms permanent deletion of an account and its content.
type DeleteUserCommand struct {
	Confirm bool `json:"confirm"`
}

type (
	// UpdateSettingsCommand updates the public URL and SMTP configuration.
	UpdateSettingsCommand Settings
	// LoginResult returns the public user and CSRF token while hiding the session cookie.
	LoginResult struct {
		User   User   `json:"user"`
		CSRF   string `json:"csrf"`
		Cookie string `json:"-"`
	}
)
