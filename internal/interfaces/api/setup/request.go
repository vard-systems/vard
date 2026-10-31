package setup

// setupRequest is the form the first administrator is made from; the address is
// the username.
type setupRequest struct {
	Email     string `json:"email" validate:"required,email,max=255"`
	Password  string `json:"password" validate:"required,max=128"`
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name" validate:"max=100"`
}
