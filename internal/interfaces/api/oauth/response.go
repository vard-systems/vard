package oauth

// errorResponse is an OAuth error (RFC 6749 section 5.2).
type errorResponse struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}
