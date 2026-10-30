package keys

// rotateRequest is the body of POST /admin/signing-keys/rotate. Both may be
// left out, for a scheduled rotation that follows the usual lead.
type rotateRequest struct {
	// Immediate makes the new keys sign now, and retires the old ones.
	Immediate bool `json:"immediate"`
	// RevokeOld deletes the old keys as well, so APIs stop trusting every
	// token they signed. It implies Immediate.
	RevokeOld bool `json:"revoke_old"`
}
