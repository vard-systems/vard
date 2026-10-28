package model

import (
	"time"

	"github.com/google/uuid"

	"vard/internal/brand"
)

// SSODomainVerification is whether a connection has proved it holds one of
// its domains. A connection signs in new people at any domain it lists, but
// links to an existing account — taking it over — only at a proved one: a
// DNS TXT record only the domain's owner could publish, or, for a domain a
// connection held before proofs existed, its being there already.
type SSODomainVerification struct {
	Base

	ConnectionID uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_sso_domain_verifications_domain,priority:1"`
	Connection   *SSOConnection `gorm:"constraint:OnDelete:CASCADE"`
	Domain       string         `gorm:"size:253;not null;uniqueIndex:idx_sso_domain_verifications_domain,priority:2"`

	// Token is the value the TXT record has to carry. It is a challenge
	// published in DNS, not a secret.
	Token string `gorm:"size:64;not null"`

	// Method is how the domain was proved: SSODomainProvedByDNS or
	// SSODomainHeldBefore. "" while it has not been.
	Method     string `gorm:"size:16;not null;default:''"`
	VerifiedAt *time.Time
	CheckedAt  *time.Time
}

func (SSODomainVerification) TableName() string {
	return "sso_domain_verifications"
}

const (
	SSODomainProvedByDNS = "dns"
	SSODomainHeldBefore  = "existing"
)

// RecordName is where the TXT record goes.
func (v SSODomainVerification) RecordName() string {
	return "_" + brand.Slug + "-challenge." + v.Domain
}

// RecordValue is what it says.
func (v SSODomainVerification) RecordValue() string {
	return brand.Slug + "-domain-verification=" + v.Token
}

func (v SSODomainVerification) Proved() bool {
	return v.VerifiedAt != nil
}
