package oidc

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"loginer/i18n"
	"loginer/internal/model"
	"loginer/internal/store"
)

// Enterprise SSO over OpenID Connect or SAML 2.0 (model.SSOConnection). This
// server is the relying party; the callback, ACS and metadata paths are given
// to the provider and must never move.
//
// What is checked:
//
//   - OIDC: PKCE and state bind the code to this sign-in; the id_token's
//     signature, issuer, audience, expiry and nonce are verified;
//   - SAML: the response must be signed by the metadata's certificate,
//     addressed to us, in time, and answer a request we sent (IdP-initiated
//     responses are refused);
//   - both: the address must be at one of the connection's domains.
const (
	PathSSOStart    = "/oauth2/sso/:slug/start"
	PathSSOCallback = "/oauth2/sso/:slug/callback"
	PathSSOACS      = "/oauth2/sso/:slug/acs"
	PathSSOMetadata = "/oauth2/sso/:slug/metadata"
)

const SSOLoginLifetime = 15 * time.Minute

var (
	ErrSSOUnknown = problem("sso_unknown", "the connection is not available")
	// ErrSSOExpired is a sign-in that took too long, came back twice, or was
	// never started here — which is also what an IdP-initiated one is.
	ErrSSOExpired        = problem("sso_expired", "the single sign-on expired or was used")
	ErrSSOUpstream       = problem("sso_upstream", "the identity provider did not complete the sign-in")
	ErrSSONoEmail        = problem("sso_no_email", "the identity provider gave no usable address")
	ErrSSODomainMismatch = problem("sso_domain_mismatch", "the address is outside the connection's domains")
	ErrSSOLinkRefused    = problem("sso_link_refused", "the address has an account the connection does not link")
	ErrSSONoAccount      = problem("sso_no_account", "the connection does not create accounts")
	ErrSSORequired       = problem("sso_required", "the address has to sign in through single sign-on")
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
		location, err = s.startSAML(connection, &login, state)
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

func (s *Service) CompleteSSOAssertion(ctx context.Context, slug string, r *http.Request, binding string, client Client) (*SocialResult, error) {
	if err := r.ParseForm(); err != nil {
		return nil, ErrSSOExpired
	}

	// No relay state is an unsolicited response: refused, whatever it says.
	connection, login, err := s.takeSSOLogin(ctx, slug, r.PostForm.Get("RelayState"), binding)
	if err != nil {
		return nil, err
	}
	if connection.Protocol != model.SSOProtocolSAML {
		return nil, ErrSSOExpired
	}

	person, err := s.completeSAML(connection, login, r)
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

func (s *Service) finishSSO(
	ctx context.Context,
	connection *model.SSOConnection,
	login *model.SSOLogin,
	person *ssoPerson,
	client Client,
) (*SocialResult, error) {
	flow, err := s.flowFor(ctx, login.Request)
	if err != nil {
		return nil, err
	}

	result, err := s.signInThroughSSO(ctx, connection, person, flow, login.Request, client)
	if err != nil {
		return nil, s.ssoFailed(ctx, connection, client, err)
	}

	return &SocialResult{SignIn: result, Request: login.Request, Next: login.Next, Provider: connection.Name}, nil
}

func (s *Service) signInThroughSSO(
	ctx context.Context,
	connection *model.SSOConnection,
	person *ssoPerson,
	flow *model.LoginFlow,
	request string,
	client Client,
) (*SignInResult, error) {
	now := s.now()
	email := strings.ToLower(strings.TrimSpace(person.Email))

	// Check a closed flow first, before any account is created or roles synced.
	if !flow.AllowSignIn {
		return nil, ErrSignInClosed
	}

	switch {
	case person.Subject == "":
		return nil, refused(ErrSSONoEmail, "the provider named nobody: no subject")
	case email == "":
		return nil, refused(ErrSSONoEmail, "no address in %s; the provider sent %s",
			strings.Join(connection.Attribute("email"), " or "), strings.Join(person.Sent, ", "))
	case !connection.OwnsEmail(email):
		return nil, refused(ErrSSODomainMismatch, "%s is outside the connection's domains", email)
	}

	var user *model.User

	identity, err := s.store.SSOIdentity(ctx, connection.ID, person.Subject)
	switch {
	case err == nil:
		if user, err = s.store.User(ctx, identity.UserID); err != nil {
			return nil, err
		}
		if err := s.store.MarkSSOIdentityUsed(ctx, identity, email, now); err != nil {
			return nil, err
		}
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	default:
		if user, err = s.ssoAccountFor(ctx, connection, person, email, client); err != nil {
			return nil, err
		}

		identity := &model.SSOIdentity{
			UserID: user.ID, ConnectionID: connection.ID, Subject: person.Subject,
			Email: email, LastLoginAt: &now,
		}
		if err := s.store.CreateSSOIdentity(ctx, identity); err != nil {
			return nil, err
		}
	}

	if !user.CanSignIn(now) {
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{
			"reason": "inactive", "connection": connection.Name,
		})
		return nil, ErrInvalidCredentials
	}

	if err := s.syncFromSSO(ctx, connection, user, person, client); err != nil {
		return nil, err
	}

	result, err := s.startSession(ctx, user, flow, request, true, client, "user.login")
	if err != nil {
		return nil, err
	}

	s.log.Info("a user signed in through single sign-on", "connection", connection.Slug, "user", user.ID)

	return result, nil
}

func (s *Service) ssoAccountFor(
	ctx context.Context,
	connection *model.SSOConnection,
	person *ssoPerson,
	email string,
	client Client,
) (*model.User, error) {
	user, err := s.store.UserByEmail(ctx, email)
	switch {
	case err == nil:
		if connection.Matching != model.SSOMatchLink {
			return nil, ErrSSOLinkRefused
		}
		// Never link to an account that has not proved its address, or whoever
		// pre-registered it would keep their password.
		if !user.IsEmailVerified {
			return nil, refused(ErrSSOLinkRefused, "the account for %s never verified its address, so it is not linked", email)
		}

		s.record(ctx, user, user.Email, "user.identity_connected", client, map[string]any{
			"provider": connection.Name,
		})

		return user, nil
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	}

	if !connection.CreateUsers {
		return nil, ErrSSONoAccount
	}

	// The provider owns the domain, so the address is as verified as it gets.
	user = &model.User{
		Email:           email,
		IsEmailVerified: true,
		FirstName:       model.Truncate(person.FirstName, 100),
		LastName:        model.Truncate(person.LastName, 100),
		IsActive:        true,
		Data:            map[string]any{},
	}

	roles, err := s.store.DefaultUserRoles(ctx)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	if err := s.store.CreateUser(ctx, user); errors.Is(err, store.ErrDuplicate) {
		return nil, ErrSSOExpired
	} else if err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, "user.registered", client, map[string]any{"provider": connection.Name})

	return user, nil
}

func (s *Service) syncFromSSO(ctx context.Context, connection *model.SSOConnection, user *model.User, person *ssoPerson, client Client) error {
	if connection.SyncProfile {
		first, last := model.Truncate(person.FirstName, 100), model.Truncate(person.LastName, 100)
		if (first != "" && first != user.FirstName) || (last != "" && last != user.LastName) {
			if first != "" {
				user.FirstName = first
			}
			if last != "" {
				user.LastName = last
			}
			if err := s.store.SaveUser(ctx, user); err != nil {
				return err
			}
		}
	}

	if len(connection.RoleMappings) == 0 {
		return nil
	}

	held := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		held = append(held, role.ID.String())
	}

	given := connection.RoleMappings.For(person.Groups)

	var add []model.UserRole
	for _, id := range given {
		if !slices.Contains(held, id.String()) {
			add = append(add, model.UserRole{ID: id})
		}
	}

	var removed []string
	if connection.SyncRoles {
		for _, id := range connection.RoleMappings.Roles() {
			if slices.Contains(held, id.String()) && !slices.Contains(given, id) {
				if err := s.store.RemoveUserRole(ctx, user.ID, id); err != nil {
					return err
				}
				removed = append(removed, id.String())
			}
		}
	}

	if len(add) > 0 {
		existing, err := s.store.UserRolesByID(ctx, rolesIDs(add))
		if err != nil {
			return err
		}
		if err := s.store.AddUserRoles(ctx, user.ID, existing); err != nil {
			return err
		}
	}

	if len(add) > 0 || len(removed) > 0 {
		s.record(ctx, user, user.Email, "user.roles_synced", client, map[string]any{
			"connection": connection.Name, "added": len(add), "removed": len(removed),
		})
	}

	return nil
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
		description = reason.message
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

func rolesIDs(roles []model.UserRole) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(roles))
	for _, role := range roles {
		out = append(out, role.ID)
	}

	return out
}
