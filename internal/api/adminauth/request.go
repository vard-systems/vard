package adminauth

// loginRequest is the body of POST /admin/auth/login. The password has no rule
// beyond being present, so the form never describes the policy to a guesser.
type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// profileRequest is the body of PATCH /admin/me. The current password is
// required only when the address changes.
type profileRequest struct {
	FirstName       string `json:"first_name" validate:"required,max=100"`
	LastName        string `json:"last_name" validate:"max=100"`
	Email           string `json:"email" validate:"required,email,max=255"`
	AvatarURL       string `json:"avatar_url"`
	CurrentPassword string `json:"current_password"`
}

type passwordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required"`
}

// codeRequest is a code from an authenticator app, or a recovery code.
type codeRequest struct {
	Code string `json:"code" validate:"required,max=32"`
}
