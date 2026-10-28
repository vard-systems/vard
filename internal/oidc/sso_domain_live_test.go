package oidc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"vard/internal/model"
)

// A connection's domain is proved by its TXT record carrying the challenge,
// and by nothing else: another value, a lookup that fails, or the record at
// another name leave it unproved.
func TestLiveADomainIsProvedByItsTXTRecordAlone(t *testing.T) {
	cases := []struct {
		name    string
		records func(name, value string) map[string][]string
		want    bool
	}{
		{"the record carries the challenge", func(name, value string) map[string][]string {
			return map[string][]string{name: {"v=spf1 -all", value}}
		}, true},
		{"the record carries another value", func(name, value string) map[string][]string {
			return map[string][]string{name: {value + "-but-another"}}
		}, false},
		{"the challenge at the bare domain, not the record name", func(_, value string) map[string][]string {
			return map[string][]string{"acme.test": {value}}
		}, false},
		{"no record at all", func(string, string) map[string][]string { return nil }, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			st := testStore(t)

			connection := &model.SSOConnection{
				Slug: "acme", Name: "Acme", Protocol: model.SSOProtocolOIDC, Issuer: "https://idp.acme.test",
				ClientID: "client", Domains: model.StringList{"acme.test"}, Matching: model.SSOMatchLink,
				Scopes: model.StringList{}, RoleMappings: model.SSORoleMappings{},
			}
			if err := st.CreateSSOConnection(ctx, connection); err != nil {
				t.Fatal(err)
			}
			challenge, err := st.SSODomainVerification(ctx, connection.ID, "acme.test")
			if err != nil {
				t.Fatal(err)
			}

			published := c.records(challenge.RecordName(), challenge.RecordValue())
			s := &Service{
				store: st, log: slog.New(slog.NewTextHandler(io.Discard, nil)), now: time.Now,
				lookupTXT: func(_ context.Context, name string) ([]string, error) {
					if records, ok := published[name]; ok {
						return records, nil
					}
					return nil, errors.New("no such host")
				},
			}

			checked, err := s.CheckSSODomain(ctx, connection.ID, "acme.test")
			if err != nil {
				t.Fatal(err)
			}
			proved, err := st.SSODomainProved(ctx, connection.ID, "acme.test")
			if err != nil {
				t.Fatal(err)
			}
			if checked.Proved() != c.want || proved != c.want || checked.CheckedAt == nil {
				t.Errorf("proved = %v (stored %v), want %v, and the check recorded", checked.Proved(), proved, c.want)
			}
		})
	}
}

// A domain the connection does not list cannot be proved for it.
func TestLiveADomainNotListedIsNotChecked(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	s := &Service{store: st, now: time.Now, lookupTXT: func(context.Context, string) ([]string, error) {
		t.Fatal("looked up a domain the connection does not list")
		return nil, nil
	}}

	connection := &model.SSOConnection{
		Slug: "acme", Name: "Acme", Protocol: model.SSOProtocolOIDC, Issuer: "https://idp.acme.test",
		ClientID: "client", Domains: model.StringList{"acme.test"}, Matching: model.SSOMatchLink,
		Scopes: model.StringList{}, RoleMappings: model.SSORoleMappings{},
	}
	if err := st.CreateSSOConnection(ctx, connection); err != nil {
		t.Fatal(err)
	}

	if _, err := s.CheckSSODomain(ctx, connection.ID, "evil.test"); !errors.Is(err, ErrSSODomainUnknown) {
		t.Errorf("checking a domain the connection does not list = %v, want ErrSSODomainUnknown", err)
	}
}
