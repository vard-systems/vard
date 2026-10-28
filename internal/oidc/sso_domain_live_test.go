package oidc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"vard/internal/database/databasetest"
	"vard/internal/model"
	"vard/internal/store"
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

// A domain carried over from before proofs links without anybody having
// proved it. Checking its record makes it proved by DNS; a check that finds
// none leaves it as it was, still linking, until it is proved or removed.
func TestLiveACarriedOverDomainIsProvedByItsRecord(t *testing.T) {
	cases := []struct {
		name        string
		found       bool
		wantMethod  string
		wantCarried bool
	}{
		{"its record is found", true, model.SSODomainProvedByDNS, false},
		{"no record is found", false, model.SSODomainHeldBefore, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			db, _ := databasetest.Open(t)
			st := store.New(db)

			connection := &model.SSOConnection{
				Slug: "acme", Name: "Acme", Protocol: model.SSOProtocolOIDC, Issuer: "https://idp.acme.test",
				ClientID: "client", Domains: model.StringList{"acme.test"}, Matching: model.SSOMatchLink,
				Scopes: model.StringList{}, RoleMappings: model.SSORoleMappings{},
			}
			if err := st.CreateSSOConnection(ctx, connection); err != nil {
				t.Fatal(err)
			}
			// What migration 00008 makes of a domain listed before proofs.
			if err := db.Exec(`UPDATE sso_domain_verifications SET method = 'existing', verified_at = now() WHERE connection_id = ?`, connection.ID).Error; err != nil {
				t.Fatal(err)
			}
			challenge, err := st.SSODomainVerification(ctx, connection.ID, "acme.test")
			if err != nil {
				t.Fatal(err)
			}

			s := &Service{store: st, now: time.Now, lookupTXT: func(_ context.Context, name string) ([]string, error) {
				if c.found && name == challenge.RecordName() {
					return []string{challenge.RecordValue()}, nil
				}
				return nil, errors.New("no such host")
			}}

			checked, err := s.CheckSSODomain(ctx, connection.ID, "acme.test")
			if err != nil {
				t.Fatal(err)
			}
			if checked.Method != c.wantMethod || checked.CarriedOver() != c.wantCarried || !checked.Proved() {
				t.Errorf("after the check: method %q, carried over %v, proved %v; want %q, %v, true",
					checked.Method, checked.CarriedOver(), checked.Proved(), c.wantMethod, c.wantCarried)
			}
			if proved, err := st.SSODomainProved(ctx, connection.ID, "acme.test"); err != nil || !proved {
				t.Errorf("the domain links = %v %v, want it still linking", proved, err)
			}
		})
	}
}
