package oidc

import (
	"context"
	"errors"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"vard/internal/model"
	"vard/internal/store"
)

// dnsTimeout bounds a domain's TXT lookup: the administrator is waiting.
const dnsTimeout = 5 * time.Second

// ErrSSODomainUnknown is a domain the connection does not list.
var ErrSSODomainUnknown = errors.New("oidc: the connection does not list this domain")

// CheckSSODomain looks for a domain's TXT record and records what it found:
// the domain is proved once the record carries its challenge.
func (s *Service) CheckSSODomain(ctx context.Context, connection uuid.UUID, domain string) (*model.SSODomainVerification, error) {
	verification, err := s.store.SSODomainVerification(ctx, connection, domain)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrSSODomainUnknown
	}
	if err != nil {
		return nil, err
	}

	lookup, cancel := context.WithTimeout(ctx, dnsTimeout)
	defer cancel()

	// A domain without the record, or whose DNS does not answer, is simply
	// not proved yet.
	records, _ := s.lookupTXT(lookup, verification.RecordName())
	proved := slices.ContainsFunc(records, func(record string) bool {
		return strings.TrimSpace(record) == verification.RecordValue()
	})

	if err := s.store.CheckedSSODomain(ctx, verification, proved, s.now()); err != nil {
		return nil, err
	}

	return verification, nil
}

func lookupTXT(ctx context.Context, name string) ([]string, error) {
	return net.DefaultResolver.LookupTXT(ctx, name)
}
