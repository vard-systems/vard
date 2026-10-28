package oidc

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vard/i18n"
	"vard/internal/model"
	"vard/internal/store"
)

// Enterprise SSO over OpenID Connect or SAML 2.0 (model.SSOConnection). This
// server is the relying party; the callback, ACS and metadata paths are given
// to the provider and must never move.
//
// What is checked:
//
//   - OIDC: PKCE and state bind the code to this sign-in; the id_token's
//     signature, issuer, audience, expiry and nonce are verified;
//   - SAML: the response must be signed by the metadata's certificate (not
//     with SHA-1), addressed to us, in time, confirm a bearer subject, and
//     answer a request we sent (IdP-initiated responses are refused); each
//     assertion is used once. The provider's cross-site POST is only held,
//     and checked on the browser's GET that follows, where its state cookie
//     arrives (HoldSSOAssertion, CompleteSSOAssertion); only the HTTP-POST
//     binding is taken;
//   - both: the address must be at one of the connection's domains.
const (
	PathSSOStart    = "/oauth2/sso/:slug/start"
	PathSSOCallback = "/oauth2/sso/:slug/callback"
	PathSSOACS      = "/oauth2/sso/:slug/acs"
	PathSSOMetadata = "/oauth2/sso/:slug/metadata"
)

const SSOLoginLifetime = 15 * time.Minute

var (
	ErrSSOUnknown = problem("sso_unknown")
	// ErrSSOExpired is a sign-in that took too long, came back twice, or was
	// never started here — which is also what an IdP-initiated one is.
	ErrSSOExpired        = problem("sso_expired")
	ErrSSOUpstream       = problem("sso_upstream")
	ErrSSONoEmail        = problem("sso_no_email")
	ErrSSODomainMismatch = problem("sso_domain_mismatch")
	ErrSSOLinkRefused    = problem("sso_link_refused")
	ErrSSONoAccount      = problem("sso_no_account")
	ErrSSORequired       = problem("sso_required")
)

type SSORequired struct {
	SSOButton
}

func (e *SSORequired) Error() string { return ErrSSORequired.Error() }
func (e *SSORequired) Unwrap() error { return ErrSSORequired }

// SSOButton is a connection as a sign-in page offers it.
type SSOButton struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

func ssoButton(connection *model.SSOConnection) SSOButton {
	return SSOButton{Slug: connection.Slug, Name: connection.Name, Protocol: string(connection.Protocol)}
}

func (s *Service) SSOButtons(ctx context.Context) ([]SSOButton, bool, error) {
	connections, err := s.store.SSOButtons(ctx)
	if err != nil {
		return nil, false, err
	}

	available, err := s.store.AnySSOConnectionEnabled(ctx)
	if err != nil {
		return nil, false, err
	}

	out := make([]SSOButton, 0, len(connections))
	for i := range connections {
		out = append(out, ssoButton(&connections[i]))
	}

	return out, available, nil
}

// DiscoverSSO returns the enabled connection owning an address's domain and
// whether it is enforced, or store.ErrNotFound. Which domains have SSO is not
// secret.
func (s *Service) DiscoverSSO(ctx context.Context, email string) (SSOButton, bool, error) {
	connection, err := s.store.SSOConnectionForEmail(ctx, email)
	if err != nil {
		return SSOButton{}, false, err
	}

	return ssoButton(connection), connection.EnforceDomains, nil
}

func (s *Service) ssoRequiredFor(ctx context.Context, email string) error {
	connection, err := s.store.SSOConnectionForEmail(ctx, email)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil
	case err != nil:
		return err
	case !connection.EnforceDomains:
		return nil
	}

	return &SSORequired{ssoButton(connection)}
}

func (s *Service) ssoConnection(ctx context.Context, slug string) (*model.SSOConnection, error) {
	connection, err := s.store.SSOConnectionBySlug(ctx, slug)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrSSOUnknown
	case err != nil:
		return nil, err
	case !connection.IsEnabled:
		return nil, ErrSSOUnknown
	}

	return connection, nil
}

func (s *Service) StartSSO(ctx context.Context, slug, request, next, loginHint string) (string, string, error) {
	connection, err := s.ssoConnection(ctx, slug)
	if err != nil {
		return "", "", err
	}

	state, stateHash, err := model.NewSecret()
	if err != nil {
		return "", "", err
	}

	login := model.SSOLogin{
		StateHash:    stateHash,
		ConnectionID: connection.ID,
		Request:      request,
		Next:         next,
		ExpiresAt:    s.now().Add(SSOLoginLifetime),
	}

	var location string

	switch connection.Protocol {
	case model.SSOProtocolOIDC:
		location, err = s.startOIDC(ctx, connection, &login, state, loginHint)
	case model.SSOProtocolSAML:
		var relay string
		if relay, login.RelayHash, err = model.NewSecret(); err == nil {
			location, err = s.startSAML(connection, &login, relay)
		}
	default:
		err = ErrSSOUnknown
	}
	if err != nil {
		return "", "", s.ssoFailed(ctx, connection, Client{}, err)
	}

	if err := s.store.CreateSSOLogin(ctx, &login); err != nil {
		return "", "", err
	}

	return location, state, nil
}

type ssoPerson struct {
	Subject   string
	Email     string
	FirstName string
	LastName  string
	Groups    []string

	Sent []string
}

func (s *Service) CompleteSSOCallback(ctx context.Context, slug, code, state, binding string, client Client) (*SocialResult, error) {
	connection, login, err := s.takeSSOLogin(ctx, slug, state, binding)
	if err != nil {
		return nil, err
	}
	if connection.Protocol != model.SSOProtocolOIDC || code == "" {
		return nil, ErrSSOExpired
	}

	person, err := s.completeOIDC(ctx, connection, login, code)
	if err != nil {
		return nil, s.ssoFailed(ctx, connection, client, err)
	}

	return s.finishSSO(ctx, connection, login, person, client)
}

// HoldSSOAssertion is the first half of the assertion consumer service: the
// provider's cross-site POST. It carries no cookie of this browser's (over
// plain http none survives it), so it decides nothing. It keeps the answer,
// sealed, on the sign-in its RelayState names, and returns the address the
// browser is sent on to, on this origin, where CompleteSSOAssertion finishes
// with the browser's own state cookie.
//
// No RelayState is an unsolicited (IdP-initiated) response: refused, whatever
// it says.
func (s *Service) HoldSSOAssertion(ctx context.Context, slug string, r *http.Request) (string, error) {
	connection, err := s.ssoConnection(ctx, slug)
	if err != nil {
		return "", err
	}
	if connection.Protocol != model.SSOProtocolSAML {
		return "", ErrSSOExpired
	}

	if err := r.ParseForm(); err != nil {
		return "", ErrSSOExpired
	}
	relay := r.PostForm.Get("RelayState")
	if relay == "" {
		return "", ErrSSOExpired
	}

	// Only the HTTP-POST binding is taken. A SAMLart would have this server
	// resolve an artifact at an address from the metadata; what was sent is
	// held as it is, and refused once the browser is known.
	relayHash := model.HashSecret(relay)
	sealed, err := s.sealer.SealFor(model.SAMLResponseWhere(relayHash), []byte(r.PostForm.Get("SAMLResponse")))
	if err != nil {
		return "", err
	}

	err = s.store.HoldSAMLResponse(ctx, connection.ID, relayHash,
		base64.RawStdEncoding.EncodeToString(sealed), s.now())
	switch {
	case errors.Is(err, store.ErrNotFound):
		return "", ErrSSOExpired
	case err != nil:
		return "", err
	}

	return connection.ACSURL(s.issuer), nil
}

// CompleteSSOAssertion is the second half: the browser, back on this origin
// with its state cookie (`binding`), takes the answer held for its own
// sign-in, which only now is checked and signs anyone in.
func (s *Service) CompleteSSOAssertion(ctx context.Context, slug, binding string, client Client) (*SocialResult, error) {
	connection, err := s.ssoConnection(ctx, slug)
	if err != nil {
		return nil, err
	}
	if connection.Protocol != model.SSOProtocolSAML || binding == "" {
		return nil, ErrSSOExpired
	}

	login, err := s.store.TakeHeldSAMLLogin(ctx, model.HashSecret(binding), s.now())
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrSSOExpired
	case err != nil:
		return nil, err
	case login.ConnectionID != connection.ID:
		return nil, ErrSSOExpired
	}

	sealed, err := base64.RawStdEncoding.DecodeString(login.SAMLResponse)
	if err != nil {
		return nil, err
	}
	response, _, err := s.sealer.OpenFor(model.SAMLResponseWhere(login.RelayHash), sealed)
	if err != nil {
		return nil, err
	}

	person, err := s.completeSAML(ctx, connection, login, string(response))
	if err != nil {
		return nil, s.ssoFailed(ctx, connection, client, err)
	}

	return s.finishSSO(ctx, connection, login, person, client)
}

func (s *Service) RefuseSSO(ctx context.Context, slug, state, binding, code, description string, client Client) error {
	connection, _, err := s.takeSSOLogin(ctx, slug, state, binding)
	if err != nil {
		return err
	}

	reason := code
	if description != "" {
		reason += ": " + description
	}

	return s.ssoFailed(ctx, connection, client, upstream("authorize", errors.New(reason)))
}

type ssoRefusal struct {
	shown *Problem
	step  string
	err   error
}

func upstream(step string, err error) error {
	return &ssoRefusal{shown: ErrSSOUpstream, step: step, err: err}
}

func refused(shown *Problem, format string, args ...any) error {
	return &ssoRefusal{shown: shown, step: "claims", err: fmt.Errorf(format, args...)}
}

func (f *ssoRefusal) Error() string { return f.step + ": " + f.err.Error() }
func (f *ssoRefusal) Unwrap() error { return f.err }

const ssoFailedAction = "sso_connection.sign_in_failed"

// ssoFailed logs the provider's error against the connection for the
// administrator, and tells the person signing in nothing of it.
func (s *Service) ssoFailed(ctx context.Context, connection *model.SSOConnection, client Client, err error) error {
	var failure *ssoRefusal
	if !errors.As(err, &failure) {
		return err
	}

	s.log.Error("an identity provider did not complete a sign-in",
		"connection", connection.Slug, "step", failure.step, "error", failure.err)

	reason := failure.err.Error()
	if len(reason) > 500 {
		reason = model.Truncate(reason, 500) + "…"
	}

	entry := model.AuditLog{
		Action:     ssoFailedAction,
		TargetType: "sso_connection",
		TargetID:   connection.ID.String(),
		IP:         client.IP,
		UserAgent:  client.UserAgent,
		Metadata: map[string]any{
			"connection": connection.Name,
			"step":       failure.step,
			"shown":      failure.shown.Code,
			"reason":     reason,
		},
	}
	if err := s.store.WriteAudit(ctx, &entry); err != nil {
		s.log.Error("writing the activity log failed", "error", err, "action", ssoFailedAction)
	}

	return failure.shown
}

// takeSSOLogin uses up the sign-in a state belongs to. `binding` must match
// what this browser kept from StartSSO, so nobody can walk another browser
// through their own sign-in.
func (s *Service) takeSSOLogin(ctx context.Context, slug, state, binding string) (*model.SSOConnection, *model.SSOLogin, error) {
	connection, err := s.ssoConnection(ctx, slug)
	if err != nil {
		return nil, nil, err
	}

	if state == "" || subtle.ConstantTimeCompare([]byte(binding), []byte(state)) != 1 {
		return nil, nil, ErrSSOExpired
	}

	login, err := s.store.TakeSSOLogin(ctx, model.HashSecret(state), s.now())
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, nil, ErrSSOExpired
	case err != nil:
		return nil, nil, err
	case login.ConnectionID != connection.ID:
		return nil, nil, ErrSSOExpired
	}

	return connection, login, nil
}

func (s *Service) SSOErrorPage(err error) string {
	reason := ErrSSOUpstream

	var known *Problem
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		reason = ErrSocialBlocked
	case errors.As(err, &known) && strings.HasPrefix(known.Code, "sso_"):
		reason = known
	}

	description, ok := i18n.Text(i18n.ID, "error."+reason.Code)
	if !ok {
		description = reason.Code
	}

	return withQuery(s.accountURL+PageError, url.Values{
		"error":             {SSOErrorCode},
		"reason":            {reason.Code},
		"error_description": {description},
	})
}

const SSOErrorCode = "sso_sign_in_failed"

func IsSSOFailure(err error) bool {
	var known *Problem
	return errors.Is(err, ErrInvalidCredentials) || errors.As(err, &known) && strings.HasPrefix(known.Code, "sso_")
}
