// Package auth signs administrators in: passwords, sessions, two-factor codes,
// and what each administrator may do.
package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"vard/internal/jose"
	"vard/internal/model"
	"vard/internal/store"
)

const SessionLifetime = 12 * time.Hour

// ErrInvalidCredentials covers an unknown user, a wrong password and a disabled
// account alike, so usernames cannot be probed.
var ErrInvalidCredentials = errors.New("invalid credentials")

var ErrNoSession = errors.New("no active session")

type State string

const (
	StateNone     State = "none"
	StateSignedIn State = "signed_in"
	StateMFA      State = "mfa"
	StateEnroll   State = "enroll"
)

const (
	challengeLifetime = 10 * time.Minute
	enrolmentLifetime = 30 * time.Minute
)

type Service struct {
	store  *store.Store
	sealer *jose.Sealer
	log    *slog.Logger

	issuer string

	// now is the clock every lifetime, lockout and expiry is measured by, so
	// a test can move it instead of waiting.
	now func() time.Time
}

func New(st *store.Store, sealer *jose.Sealer, log *slog.Logger, issuer string) *Service {
	return &Service{store: st, sealer: sealer, log: log, issuer: issuer, now: time.Now}
}

// MFARequired reports whether every administrator must use a second factor.
// When the setting cannot be read the answer is yes.
func (s *Service) MFARequired(ctx context.Context) bool {
	security, err := s.store.AdminSecurity(ctx)
	if err != nil {
		s.log.Error("reading the admin security settings failed", "error", err)
		return true
	}

	return security.RequireMFA
}

type Request struct {
	IP        string
	UserAgent string
}

func (s *Service) Login(ctx context.Context, username, password string, req Request) (string, *model.Admin, State, error) {
	admin, err := s.store.AdminByUsername(ctx, username)

	switch {
	case errors.Is(err, store.ErrNotFound):
		// Still hash something, so a missing username and a wrong password
		// take the same time to answer.
		model.CompareDummyPassword(password)
		s.record(ctx, nil, username, "admin.login_failed", req, "unknown username")
		return "", nil, StateNone, ErrInvalidCredentials
	case err != nil:
		return "", nil, StateNone, err
	}

	// The attempt is reserved before comparing, so parallel guesses cannot pass
	// a lock, and a locked account takes as long to refuse as a wrong password.
	now := s.now()
	allowed, err := s.store.ReserveAdminLogin(ctx, admin, now, model.MaxFailedLogins, model.LockoutDuration)
	if err != nil {
		return "", nil, StateNone, err
	}
	if !allowed {
		model.CompareDummyPassword(password)
		s.record(ctx, &admin.ID, admin.Username, "admin.login_blocked", req, "locked")

		return "", nil, StateNone, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		s.record(ctx, &admin.ID, admin.Username, "admin.login_failed", req, "wrong password")

		return "", nil, StateNone, ErrInvalidCredentials
	}

	if admin.Status != model.StatusActive {
		s.record(ctx, &admin.ID, admin.Username, "admin.login_blocked", req, string(admin.Status))

		return "", nil, StateNone, ErrInvalidCredentials
	}

	state, lifetime := StateSignedIn, SessionLifetime
	switch {
	case admin.HasMFA():
		state, lifetime = StateMFA, challengeLifetime
	case s.MFARequired(ctx):
		state, lifetime = StateEnroll, enrolmentLifetime
	}

	// Codes are counted against the same attempts, so while one is owed the
	// password must not give them back: signing in again between guesses
	// would otherwise make the lockout unreachable. The code itself clears
	// them (signedIn).
	if state != StateMFA {
		if err := s.store.ClearAdminFailedLogins(ctx, admin); err != nil {
			return "", nil, StateNone, err
		}
	}

	token, hash, err := model.NewSecret()
	if err != nil {
		return "", nil, StateNone, err
	}

	session := model.AdminSession{
		AdminID:     admin.ID,
		TokenHash:   hash,
		ExpiresAt:   s.now().Add(lifetime),
		IsMFAPassed: state == StateSignedIn,
		UserAgent:   model.Truncate(req.UserAgent, 255),
		IP:          model.Truncate(req.IP, 45),
	}
	if err := s.store.CreateAdminSession(ctx, &session); err != nil {
		return "", nil, StateNone, err
	}

	if state == StateSignedIn {
		if err := s.signedIn(ctx, admin, req, ""); err != nil {
			return "", nil, StateNone, err
		}
	}

	return token, admin, state, nil
}

func (s *Service) signedIn(ctx context.Context, admin *model.Admin, req Request, method string) error {
	if err := s.store.MarkAdminSignedIn(ctx, admin, s.now(), req.IP); err != nil {
		return err
	}

	s.recordWith(ctx, &admin.ID, admin.Username, "admin.login", req, map[string]any{"second_factor": method})

	return nil
}

func (s *Service) Session(ctx context.Context, token string) (*model.Admin, *model.AdminSession, State, error) {
	if token == "" {
		return nil, nil, StateNone, ErrNoSession
	}

	session, err := s.store.AdminSessionByTokenHash(ctx, model.HashSecret(token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, nil, StateNone, ErrNoSession
	case err != nil:
		return nil, nil, StateNone, err
	}

	now := s.now()
	if session.RevokedAt != nil || !now.Before(session.ExpiresAt) {
		return nil, nil, StateNone, ErrNoSession
	}

	// The lockout applies to signing in, not to open sessions; otherwise five
	// wrong passwords from anyone would sign an administrator out.
	admin, err := s.store.AdminPrincipal(ctx, session.AdminID)
	if err != nil || admin.Status != model.StatusActive {
		return nil, nil, StateNone, ErrNoSession
	}

	switch {
	case session.IsMFAPassed && s.MFARequired(ctx) && !admin.HasMFA():
		// Signed in before two-factor sign-in was required, or had it reset:
		// the session is good for setting it up and nothing else.
		return admin, session, StateEnroll, nil
	case session.IsMFAPassed:
		return admin, session, StateSignedIn, nil
	case admin.HasMFA():
		return admin, session, StateMFA, nil
	case s.MFARequired(ctx):
		return admin, session, StateEnroll, nil
	default:
		return nil, nil, StateNone, ErrNoSession
	}
}

func (s *Service) Authenticate(ctx context.Context, token string) (*model.Admin, *model.AdminSession, error) {
	admin, session, state, err := s.Session(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	if state != StateSignedIn {
		return nil, nil, ErrNoSession
	}

	return admin, session, nil
}

func (s *Service) Logout(ctx context.Context, token string, req Request) error {
	if token == "" {
		return nil
	}

	session, err := s.store.AdminSessionByTokenHash(ctx, model.HashSecret(token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil
	case err != nil:
		return err
	}

	if err := s.store.RevokeAdminSession(ctx, session, s.now()); err != nil {
		return err
	}

	if admin, err := s.store.AdminByID(ctx, session.AdminID); err == nil {
		s.record(ctx, &admin.ID, admin.Username, "admin.logout", req, "")
	}

	return nil
}

func (s *Service) record(ctx context.Context, adminID *uuid.UUID, actor, action string, req Request, note string) {
	var metadata map[string]any
	if note != "" {
		metadata = map[string]any{"reason": note}
	}

	s.recordWith(ctx, adminID, actor, action, req, metadata)
}

func (s *Service) recordWith(ctx context.Context, adminID *uuid.UUID, actor, action string, req Request, metadata map[string]any) {
	entry := model.AuditLog{
		AdminID:    adminID,
		ActorEmail: actor,
		Action:     action,
		TargetType: "admin_user",
		IP:         req.IP,
		UserAgent:  req.UserAgent,
	}
	for key, value := range metadata {
		if value != "" && value != nil {
			if entry.Metadata == nil {
				entry.Metadata = map[string]any{}
			}
			entry.Metadata[key] = value
		}
	}
	if adminID != nil {
		entry.TargetID = adminID.String()
	}

	// The action itself has happened; a lost log line must not undo it, but
	// it must not go unnoticed either.
	if err := s.store.WriteAudit(ctx, &entry); err != nil {
		s.log.Error("writing the activity log failed", "error", err, "action", action)
	}
}
