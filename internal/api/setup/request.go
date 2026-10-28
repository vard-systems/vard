package setup

// setupRequest is the form the first administrator is made from; the address is
// the username.
type setupRequest struct {
	Email     string `json:"email" validate:"required,email,max=255"`
	Password  string `json:"password" validate:"required,min=10,max=128"`
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name" validate:"max=100"`
	// SetupToken is the one the server logged at startup.
	SetupToken string `json:"setup_token" validate:"max=64"`
}
