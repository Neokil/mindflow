package authentication

// UserDelivery pairs account metadata with the latest mail-delivery failure.
type UserDelivery struct {
	User      User   `json:"user"`
	MailError string `json:"mailError"`
}
