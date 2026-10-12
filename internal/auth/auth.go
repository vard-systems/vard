// Package auth signs administrators in and out, with a password and an optional
// or required second factor.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"loginer/internal/jose"
	"loginer/internal/model"
	"loginer/internal/store"
)

// SessionLifetime is how long a session lasts before the administrator has to
// sign in again.
const SessionLifetime = 12 * time.Hour

// MaxFailedLogins wrong passwords in a row lock an account for LockoutDuration:
// long enough to stop guessing, short enough that an attacker cannot keep an
// administrator out.
const (
	MaxFailedLogins = 5
	LockoutDuration = 15 * time.Minute
)

// dummyHash is compared for unknown usernames so they take as long as a wrong
// password. It must be a real hash at full cost.
var dummyHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("not a password anyone has"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("auth: make the dummy hash: %v", err))
	}

	return hash
})

// ErrInvalidCredentials covers an unknown user, a wrong password and a disabled
// account alike, so usernames cannot be probed.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrNoSession is returned when a request carries no usable session.
var ErrNoSession = errors.New("no active session")

// State is how far a session has got.
type State string

const (
	// StateNone is no usable session at all.
	StateNone State = "none"
	// StateSignedIn is a session that may use the panel.
	StateSignedIn State = "signed_in"
	// StateMFA is a session whose password was right, waiting for a code
	// from the administrator's authenticator. It can do nothing but give one.
	StateMFA State = "mfa"
	// StateEnroll is a password-verified session that must set up a second
	// factor before anything else.
	StateEnroll State = "enroll"
)

// How long a session may wait half signed in.
const (
	challengeLifetime = 10 * time.Minute
	enrolmentLifetime = 30 * time.Minute
)

// Service signs administrators in and out.
type Service struct {
	store  *store.Store
	sealer *jose.Sealer
	log    *slog.Logger

	// issuer names the panel in authenticator apps.
	issuer string
}

// New returns a Service. `sealer` encrypts TOTP secrets and `issuer` is what
// authenticator apps list the account under. Whether MFA is required is read
// from the database each time, so a change applies on the next sign-in.
func New(st *store.Store, sealer *jose.Sealer, log *slog.Logger, issuer string) *Service {
	return &Service{store: st, sealer: sealer, log: log, issuer: issuer}
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

// Request describes where a call came from, which is recorded on the session
// and in the log.
type Request struct {
	IP        string
	UserAgent string
}

// Login checks the credentials and starts a session. The returned token is the
// only copy; the database keeps its hash. An administrator who still owes a
// second factor gets a session in StateMFA or StateEnroll that can do nothing
// else.
func (s *Service) Login(ctx context.Context, username, password string, req Request) (string, *model.Admin, State, error) {
	admin, err := s.store.AdminByUsername(ctx, username)

	switch {
	case errors.Is(err, store.ErrNotFound):
		// Still hash something, so a missing username and a wrong password
		// take the same time to answer.
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		s.record(ctx, nil, username, "admin.login_failed", req, "unknown username")
		return "", nil, StateNone, ErrInvalidCredentials
	case err != nil:
		return "", nil, StateNone, err
	}

	// The attempt is reserved before comparing, so parallel guesses cannot pass
	// a lock, and a locked account takes as long to refuse as a wrong password.
	now := time.Now()
	allowed, err := s.store.ReserveAdminLogin(ctx, admin, now, MaxFailedLogins, LockoutDuration)
	if err != nil {
		return "", nil, StateNone, err
	}
	if !allowed {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
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

	token, err := newToken()
	if err != nil {
		return "", nil, StateNone, err
	}

	session := model.AdminSession{
		AdminID:     admin.ID,
		TokenHash:   hashToken(token),
		ExpiresAt:   time.Now().Add(lifetime),
		IsMFAPassed: state == StateSignedIn,
		UserAgent:   model.Truncate(req.UserAgent, 255),
		IP:          model.Truncate(req.IP, 45),
	}
	if err := s.store.CreateSession(ctx, &session); err != nil {
		return "", nil, StateNone, err
	}

	if state == StateSignedIn {
		if err := s.signedIn(ctx, admin, req, ""); err != nil {
			return "", nil, StateNone, err
		}
	}

	return token, admin, state, nil
}

// signedIn finishes a sign-in: the last sign-in is recorded, the wrong
// passwords before it forgotten, and the log told how it happened.
func (s *Service) signedIn(ctx context.Context, admin *model.Admin, req Request, method string) error {
	if err := s.store.MarkAdminSignedIn(ctx, admin, time.Now(), req.IP); err != nil {
		return err
	}

	s.recordWith(ctx, &admin.ID, admin.Username, "admin.login", req, map[string]any{"second_factor": method})

	return nil
}

// Session returns the session a token carries, whatever state it is in, with
// its administrator.
func (s *Service) Session(ctx context.Context, token string) (*model.Admin, *model.AdminSession, State, error) {
	if token == "" {
		return nil, nil, StateNone, ErrNoSession
	}

	session, err := s.store.SessionByTokenHash(ctx, hashToken(token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, nil, StateNone, ErrNoSession
	case err != nil:
		return nil, nil, StateNone, err
	}

	now := time.Now()
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
		// Waiting for a factor that has since been removed, where none is
		// required: sign in again.
		return nil, nil, StateNone, ErrNoSession
	}
}

// Authenticate returns the administrator and session for a fully signed-in
// token; a half-signed-in session is refused.
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

// Logout revokes the session the token belongs to. Signing out twice is not
// an error.
func (s *Service) Logout(ctx context.Context, token string, req Request) error {
	if token == "" {
		return nil
	}

	session, err := s.store.SessionByTokenHash(ctx, hashToken(token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil
	case err != nil:
		return err
	}

	if err := s.store.RevokeSession(ctx, session, time.Now()); err != nil {
		return err
	}

	if admin, err := s.store.AdminByID(ctx, session.AdminID); err == nil {
		s.record(ctx, &admin.ID, admin.Username, "admin.logout", req, "")
	}

	return nil
}

// record writes to the activity log; a failure is logged but never fails the
// request.
func (s *Service) record(ctx context.Context, adminID *uuid.UUID, actor, action string, req Request, note string) {
	var metadata map[string]any
	if note != "" {
		metadata = map[string]any{"reason": note}
	}

	s.recordWith(ctx, adminID, actor, action, req, metadata)
}

// recordWith is record with metadata of the caller's choosing. Empty values are
// left out.
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

// newToken returns a session token: 256 bits of randomness, hex encoded.
func newToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// hashToken is what gets stored. SHA-256 is right here, unlike for passwords:
// the token is long and random, so it cannot be guessed by brute force.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
