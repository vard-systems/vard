package otp

// What this page's record is called in the activity log. There is one of it,
// so the target is the page rather than a row's id.
const (
	targetType = "otp"
	targetID   = "settings"
)

// settingsRequest is a PATCH: nil leaves a setting alone.
type settingsRequest struct {
	CodeLength      *int `json:"code_length"`
	LifetimeMinutes *int `json:"lifetime_minutes"`
	MaxAttempts     *int `json:"max_attempts"`
	ResendSeconds   *int `json:"resend_seconds"`
}
