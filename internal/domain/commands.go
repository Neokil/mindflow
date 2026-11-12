package domain

type CreateWorkspaceCommand struct {
	Name string `json:"name"`
}
type RenameWorkspaceCommand struct {
	Name     string `json:"name"`
	Revision int    `json:"revision"`
}
type DeleteWorkspaceCommand struct {
	Revision int  `json:"revision"`
	Confirm  bool `json:"confirm"`
}
type CreatePageCommand struct {
	Name string `json:"name"`
}
type RenamePageCommand struct {
	Name     string `json:"name"`
	Revision int    `json:"revision"`
}
type DeletePageCommand struct {
	Revision int  `json:"revision"`
	Confirm  bool `json:"confirm"`
}
type ReorderPagesCommand struct {
	IDs      []string `json:"ids"`
	Revision int      `json:"revision"`
}
type SaveDocumentCommand struct {
	BaseRevision int      `json:"baseRevision"`
	Epoch        int64    `json:"epoch"`
	ClientID     string   `json:"clientId"`
	MutationID   string   `json:"mutationId"`
	Document     Document `json:"document"`
}
type RecoveryCopyCommand struct {
	WorkspaceID string   `json:"workspaceId"`
	Name        string   `json:"name"`
	Document    Document `json:"document"`
}
type LeaseCommand struct {
	ClientID string `json:"clientId"`
	Action   string `json:"action"`
	Epoch    int64  `json:"epoch"`
}
type SetupCommand struct {
	Credential string   `json:"credential"`
	Email      string   `json:"email"`
	Password   string   `json:"password"`
	Settings   Settings `json:"settings"`
}
type SetupTestEmailCommand struct {
	Credential string   `json:"credential"`
	Email      string   `json:"email"`
	Settings   Settings `json:"settings"`
}
type LoginCommand struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type InviteCommand struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}
type ForgotCommand struct {
	Email string `json:"email"`
}
type RedeemCommand struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
type UpdateUserCommand struct {
	Role   string `json:"role"`
	Status string `json:"status"`
}
type DeleteUserCommand struct {
	Confirm bool `json:"confirm"`
}
type UpdateSettingsCommand Settings
type RequestVariantCommand struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
type LoginResult struct {
	User   User   `json:"user"`
	CSRF   string `json:"csrf"`
	Cookie string `json:"-"`
}
