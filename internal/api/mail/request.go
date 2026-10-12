package mail

// What this page's records are called in the activity log. There is one of
// each, so the target is the page rather than a row's id.
const (
	targetType = "mail"
	targetID   = "settings"

	contentTargetType = "mail_content"
)

// settingsRequest is a PATCH: nil leaves a setting alone. The panel never
// receives the stored password, so omitting it keeps it and an empty string
// clears it.
type settingsRequest struct {
	IsEnabled   *bool   `json:"is_enabled"`
	Host        *string `json:"host"`
	Port        *int    `json:"port"`
	Encryption  *string `json:"encryption"`
	Username    *string `json:"username"`
	Password    *string `json:"password"`
	FromAddress *string `json:"from_address"`
	FromName    *string `json:"from_name"`
}

// testRequest is a test send with the form's settings, falling back to stored
// values.
type testRequest struct {
	To string `json:"to"`

	settingsRequest
}

// contentRequest is one language's email text. An empty value clears the
// override and brings the shipped text back.
type contentRequest struct {
	Messages map[string]string `json:"messages"`
}
