package account

// loginRequest is the body of POST /account/login. Request is the handle of
// the sign-in under way, when there is one.
type loginRequest struct {
	Request  string `json:"request" validate:"max=64"`
	Email    string `json:"email" validate:"required,max=255"`
	Password string `json:"password" validate:"required,max=128"`
	// Remember is the "stay signed in" box. A flow that does not offer it
	// ignores whatever is sent.
	Remember bool `json:"remember"`
}

type registerRequest struct {
	Request     string `json:"request" validate:"required,max=64"`
	Email       string `json:"email" validate:"required,email,max=255"`
	Password    string `json:"password" validate:"required,max=72"`
	FirstName   string `json:"first_name" validate:"max=100"`
	LastName    string `json:"last_name" validate:"max=100"`
	AcceptTerms bool   `json:"accept_terms"`
	Remember    bool   `json:"remember"`
}

// codeRequest is the body of POST /account/login/code: the handle the sign-in
// was held under, and the code from the message.
type codeRequest struct {
	Handle string `json:"handle" validate:"required,max=64"`
	Code   string `json:"code" validate:"required,max=16"`
}

type resendRequest struct {
	Handle string `json:"handle" validate:"required,max=64"`
}

// discoverRequest is the address "Sign in with SSO" looks a connection up by.
type discoverRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
}

type forgotRequest struct {
	Request string `json:"request" validate:"max=64"`
	Email   string `json:"email" validate:"required,email,max=255"`
	// Language is the one the page was shown in, which the email is written
	// in. One that is not offered, or none, is the default.
	Language string `json:"language" validate:"max=16"`
}

// resetRequest is the body of POST /account/reset-password.
// verifyRequest is the body of POST /account/verify-email.
type verifyRequest struct {
	Token string `json:"token" validate:"required,max=64"`
}

type resetRequest struct {
	Token    string `json:"token" validate:"required,max=64"`
	Password string `json:"password" validate:"required,max=72"`
}

type profileRequest struct {
	FirstName string `json:"first_name" validate:"max=100"`
	LastName  string `json:"last_name" validate:"max=100"`
}

// emailRequest is the body of POST /account/email: the address to move to.
type emailRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
	// CurrentPassword proves the person at the session owns the account; a
	// wrong one counts towards the lockout, as at sign-in.
	CurrentPassword string `json:"current_password" validate:"required,max=128"`
}

type passwordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,max=128"`
	NewPassword     string `json:"new_password" validate:"required,max=72"`
}
