package profile

import (
	"time"

	"github.com/google/uuid"

	"vard/internal/model"
)

// userResponse is built by hand so new columns, roles and organisation fields
// are never published to the user by accident.
type userResponse struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at"`
}

func newUserResponse(user *model.User) userResponse {
	return userResponse{
		ID:            user.ID,
		Email:         user.Email,
		EmailVerified: user.IsEmailVerified,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		CreatedAt:     user.CreatedAt,
		LastLoginAt:   user.LastLoginAt,
	}
}
