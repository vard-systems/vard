package profile

type profileRequest struct {
	FirstName string `json:"first_name" validate:"max=100"`
	LastName  string `json:"last_name" validate:"max=100"`
}

// emailRequest is the body of POST /account/email: the address to move to.
type emailRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
	// CurrentPassword proves the person at the session owns the account; a
	// wrong one counts towards the throttle, as at sign-in.
	CurrentPassword string `json:"current_password" validate:"required,max=128"`
}

type passwordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,max=128"`
	NewPassword     string `json:"new_password" validate:"required,max=128"`
}
