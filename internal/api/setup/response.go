package setup

import "vard/internal/model"

// statusResponse says whether the panel still needs an administrator.
type statusResponse struct {
	Required bool `json:"required"`
}

// adminResponse is the account that was just made, as the panel sees it.
type adminResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

func newAdminResponse(a *model.Admin) adminResponse {
	return adminResponse{ID: a.ID.String(), Email: a.Email, FullName: a.FullName()}
}
