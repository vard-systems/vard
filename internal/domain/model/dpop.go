package model

import "time"

// DPoPProofWindow is how far a DPoP proof's iat may be from now, either way,
// and how long its jti is remembered: long enough that no proof can be used
// twice while it would still be accepted.
const DPoPProofWindow = 2 * time.Minute

// DPoPProof is a DPoP proof already used (RFC 9449 §11.1): its key's
// thumbprint and jti, hashed, so the same proof is never accepted twice.
type DPoPProof struct {
	Base

	Hash      string    `gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (DPoPProof) TableName() string {
	return "dpop_proofs"
}
