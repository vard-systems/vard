package store

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loginer/internal/brand"
	"loginer/internal/cache"
	"loginer/internal/model"
)

// ErrAlreadyUsed is a single-use secret presented again; it differs from
// ErrNotFound because a replay revokes what the first use issued.
var ErrAlreadyUsed = errors.New("already used")

func (s *Store) SigningKeys(ctx context.Context) ([]model.SigningKey, error) {
	var keys []model.SigningKey
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&keys).Error

	return keys, err
}

func (s *Store) CreateSigningKey(ctx context.Context, key *model.SigningKey) error {
	return translate(s.db.WithContext(ctx).Create(key).Error)
}

func (s *Store) ApplicationByClientID(ctx context.Context, clientID string) (*model.Application, error) {
	kept, err := cached(ctx, s, cache.Clients, "client_id:"+clientID, func() (client, error) {
		var app model.Application
		if err := s.db.WithContext(ctx).First(&app, "client_id = ?", clientID).Error; err != nil {
			return client{}, translate(err)
		}

		return client{Application: app, SecretHash: app.ClientSecretHash}, nil
	})
	if err != nil {
		return nil, err
	}

	app := kept.Application
	app.ClientSecretHash = kept.SecretHash

	return &app, nil
}

// client is how an application is cached, including its secret hash (a SHA-256
// of a random secret) which the model omits from JSON.
type client struct {
	Application model.Application `json:"application"`
	SecretHash  string            `json:"secret_hash"`
}

type Audience struct {
	API        *model.API
	Authorized bool
	Allowed    []string
}

func (s *Store) AudienceFor(ctx context.Context, applicationID uuid.UUID, identifier string) (Audience, error) {
	return cached(ctx, s, cache.Grants, "audience:"+applicationID.String()+":"+identifier, func() (Audience, error) {
		return s.audienceFor(ctx, applicationID, identifier)
	})
}

func (s *Store) audienceFor(ctx context.Context, applicationID uuid.UUID, identifier string) (Audience, error) {
	api, err := s.APIByIdentifier(ctx, identifier)
	if err != nil {
		return Audience{}, err
	}

	var authorized int64
	err = s.db.WithContext(ctx).Model(&model.ApplicationAPI{}).
		Where("application_id = ? AND api_id = ?", applicationID, api.ID).Count(&authorized).Error
	if err != nil {
		return Audience{}, err
	}

	var allowed []uuid.UUID
	err = s.db.WithContext(ctx).Model(&model.ApplicationAPIScope{}).
		Joins("JOIN api_scopes ON api_scopes.id = application_api_scopes.api_scope_id").
		Where("application_api_scopes.application_id = ? AND api_scopes.api_id = ?", applicationID, api.ID).
		Pluck("application_api_scopes.api_scope_id", &allowed).Error
	if err != nil {
		return Audience{}, err
	}

	out := Audience{API: api, Authorized: authorized > 0, Allowed: []string{}}
	for _, scope := range api.Scopes {
		if slices.Contains(allowed, scope.ID) {
			out.Allowed = append(out.Allowed, scope.Name)
		}
	}

	return out, nil
}

func (s *Store) EffectiveRoles(ctx context.Context, user *model.User) ([]model.UserRole, error) {
	graph, err := s.RoleGraph(ctx)
	if err != nil {
		return nil, err
	}

	held := make([]uuid.UUID, 0, len(user.Roles))
	for _, role := range user.Roles {
		held = append(held, role.ID)
	}

	return graph.Effective(held), nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := s.db.WithContext(ctx).Preload("Roles", byName).First(&user, "email = ?", model.NormalizeEmail(email)).Error
	if err != nil {
		return nil, translate(err)
	}

	return &user, nil
}

// MarkUserSignedIn records the sign-in and clears failed attempts. It writes
// only these columns, so roles removed by SSO role sync during the sign-in are
// not written back.
func (s *Store) MarkUserSignedIn(ctx context.Context, user *model.User, at time.Time) error {
	return s.db.WithContext(ctx).Model(user).Omit(clause.Associations).Updates(map[string]any{
		"last_login_at":      at,
		"failed_login_count": 0,
		"locked_until":       nil,
	}).Error
}

func (s *Store) ReserveUserLogin(ctx context.Context, user *model.User, at time.Time, max int, lockFor time.Duration) (bool, error) {
	var rows []struct {
		LockedUntil *time.Time
	}

	// The WHERE matches no row while locked, so nothing is counted; otherwise
	// the attempt is counted and the `max`th locks.
	err := s.db.WithContext(ctx).Raw(`
		UPDATE users SET
			locked_until = CASE WHEN failed_login_count + 1 >= @max THEN @until ELSE locked_until END,
			failed_login_count = CASE WHEN failed_login_count + 1 >= @max THEN 0 ELSE failed_login_count + 1 END
		WHERE id = @id AND (locked_until IS NULL OR locked_until <= @now)
		RETURNING locked_until`,
		map[string]any{"max": max, "until": at.Add(lockFor), "id": user.ID, "now": at},
	).Scan(&rows).Error
	if err != nil {
		return false, err
	}

	return len(rows) > 0, nil
}

func (s *Store) ClearUserFailedLogins(ctx context.Context, user *model.User, at time.Time) error {
	return s.db.WithContext(ctx).Model(user).Omit(clause.Associations).Updates(map[string]any{
		"failed_login_count": 0,
		"locked_until":       nil,
	}).Error
}

func (s *Store) CreateAuthorizationRequest(ctx context.Context, req *model.AuthorizationRequest) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(req).Error)
}

func (s *Store) AuthorizationRequestByHandle(ctx context.Context, hash string) (*model.AuthorizationRequest, error) {
	var req model.AuthorizationRequest
	err := s.db.WithContext(ctx).Preload("Application").First(&req, "handle_hash = ?", hash).Error
	if err != nil {
		return nil, translate(err)
	}

	return &req, nil
}

// CompleteAuthorizationRequest marks a sign-in finished; only the first call
// succeeds, so a handle yields one code.
func (s *Store) CompleteAuthorizationRequest(ctx context.Context, req *model.AuthorizationRequest, at time.Time) error {
	result := s.db.WithContext(ctx).Model(&model.AuthorizationRequest{}).
		Where("id = ? AND completed_at IS NULL", req.ID).
		Update("completed_at", at)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyUsed
	}

	req.CompletedAt = &at

	return nil
}

func (s *Store) CreateAuthorizationCode(ctx context.Context, code *model.AuthorizationCode) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(code).Error)
}

// ClaimAuthorizationCode marks this application's code used and returns it. A
// concurrent second claim, or a code belonging to another application, gets
// ErrAlreadyUsed with the code, which is never written to, so a client that saw
// someone else's code cannot burn it.
func (s *Store) ClaimAuthorizationCode(ctx context.Context, hash string, application uuid.UUID, at time.Time) (*model.AuthorizationCode, error) {
	var codes []model.AuthorizationCode
	err := s.db.WithContext(ctx).Model(&codes).
		Clauses(clause.Returning{}).
		Where("code_hash = ? AND application_id = ? AND used_at IS NULL", hash, application).
		Update("used_at", at).Error
	if err != nil {
		return nil, err
	}
	if len(codes) == 1 {
		return &codes[0], nil
	}

	var used model.AuthorizationCode
	if err := s.db.WithContext(ctx).First(&used, "code_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &used, ErrAlreadyUsed
}

func (s *Store) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(token).Error)
}

func (s *Store) RefreshTokenByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	if err := s.db.WithContext(ctx).First(&token, "token_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &token, nil
}

// RotateRefreshToken revokes `old` and stores `next` in one transaction; if
// `old` was already revoked it returns ErrAlreadyUsed.
func (s *Store) RotateRefreshToken(ctx context.Context, old, next *model.RefreshToken, at time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.RefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", old.ID).
			Update("revoked_at", at)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAlreadyUsed
		}

		return translate(tx.Omit(clause.Associations).Create(next).Error)
	})
}

func (s *Store) RevokeRefreshFamily(ctx context.Context, family uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", family).
		Update("revoked_at", at).Error
}

func (s *Store) RevokeRefreshTokensForCode(ctx context.Context, code uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("code_id = ? AND revoked_at IS NULL", code).
		Update("revoked_at", at).Error
}

func (s *Store) RevokeRefreshTokensForUser(ctx context.Context, user uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", user).
		Update("revoked_at", at).Error
}

func (s *Store) CreateUserSession(ctx context.Context, session *model.UserSession) error {
	if err := translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(session).Error); err != nil {
		return err
	}

	s.putUserSessions(ctx, *session)

	return nil
}

func (s *Store) UserSessionByHash(ctx context.Context, hash string) (*model.UserSession, error) {
	var kept userSession
	if s.sessions.Session(ctx, cache.UserSession, hash, &kept) {
		session := kept.model(hash)
		return &session, nil
	}

	var session model.UserSession
	if err := s.db.WithContext(ctx).First(&session, "token_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	s.sessions.FillSession(ctx, cache.UserSession, hash, newUserSession(session), session.ExpiresAt)

	return &session, nil
}

func (s *Store) RevokeUserSession(ctx context.Context, id uuid.UUID, at time.Time) error {
	ended, err := revokeUserSessionsIn(s.db.WithContext(ctx), at, "id = ?", id)
	s.putUserSessions(ctx, ended...)

	return err
}

func (s *Store) RevokeUserSessionsFor(ctx context.Context, user uuid.UUID, at time.Time) error {
	ended, err := revokeUserSessionsIn(s.db.WithContext(ctx), at, "user_id = ?", user)
	s.putUserSessions(ctx, ended...)

	return err
}

// pendingEmailChangesUsed cancels pending address changes wherever access is
// reset (password reset or change, sign out everywhere); otherwise whoever
// requested one could take the account back.
const pendingEmailChangesUsed = "UPDATE email_verifications SET used_at = @at " +
	"WHERE user_id = @user AND used_at IS NULL AND new_email <> ''"

func (s *Store) CreatePasswordReset(ctx context.Context, reset *model.PasswordReset) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(reset).Error)
}

const passwordResetsPerHour = 5

// CreatePasswordResetThrottled stores a reset link only if the account got none
// in the last minute and fewer than passwordResetsPerHour in the hour. A
// per-account advisory lock stops a burst from all passing the check.
func (s *Store) CreatePasswordResetThrottled(ctx context.Context, reset *model.PasswordReset, now time.Time) (bool, error) {
	created := false

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", reset.UserID.String()).Error; err != nil {
			return translate(err)
		}

		var lastMinute, lastHour int64
		if err := tx.Model(&model.PasswordReset{}).
			Where("user_id = ? AND created_at > ?", reset.UserID, now.Add(-time.Minute)).
			Count(&lastMinute).Error; err != nil {
			return translate(err)
		}
		if err := tx.Model(&model.PasswordReset{}).
			Where("user_id = ? AND created_at > ?", reset.UserID, now.Add(-time.Hour)).
			Count(&lastHour).Error; err != nil {
			return translate(err)
		}
		if lastMinute > 0 || lastHour >= passwordResetsPerHour {
			return nil
		}

		created = true

		return translate(tx.Omit(clause.Associations).Create(reset).Error)
	})

	return created, err
}

func (s *Store) PasswordResetByHash(ctx context.Context, hash string) (*model.PasswordReset, error) {
	var reset model.PasswordReset
	if err := s.db.WithContext(ctx).First(&reset, "token_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &reset, nil
}

func (s *Store) ResetPassword(ctx context.Context, reset *model.PasswordReset, user *model.User, at time.Time) error {
	var ended []model.UserSession

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.PasswordReset{}).
			Where("id = ? AND used_at IS NULL", reset.ID).
			Update("used_at", at)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAlreadyUsed
		}

		changes := map[string]any{
			"password_hash":         user.PasswordHash,
			"is_password_temporary": false,
			"failed_login_count":    0,
			"locked_until":          nil,
		}
		// An emailed link proves the address; the link a temporary password
		// hands out does not.
		if !user.IsPasswordTemporary {
			changes["is_email_verified"] = true
		}
		err := tx.Model(user).Updates(changes).Error
		if err != nil {
			return err
		}

		if ended, err = revokeUserSessionsIn(tx, at, "user_id = ?", user.ID); err != nil {
			return err
		}

		statements := []string{
			"UPDATE refresh_tokens SET revoked_at = @at WHERE user_id = @user AND revoked_at IS NULL",
			"UPDATE password_resets SET used_at = @at WHERE user_id = @user AND used_at IS NULL",
			pendingEmailChangesUsed,
		}
		for _, statement := range statements {
			if err := tx.Exec(statement, map[string]any{"at": at, "user": user.ID}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.putUserSessions(ctx, ended...)

	return nil
}

func (s *Store) ActiveUserSessions(ctx context.Context, user uuid.UUID, now time.Time) ([]model.UserSession, error) {
	var sessions []model.UserSession
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", user, now).
		Order("authenticated_at DESC").
		Find(&sessions).Error

	return sessions, err
}

func (s *Store) ChangeUserPassword(ctx context.Context, user *model.User, keep uuid.UUID, at time.Time) error {
	var ended []model.UserSession

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(user).Updates(map[string]any{
			"password_hash":         user.PasswordHash,
			"is_password_temporary": false,
		}).Error
		if err != nil {
			return err
		}

		if ended, err = revokeUserSessionsIn(tx, at, "user_id = ? AND id <> ?", user.ID, keep); err != nil {
			return err
		}

		statements := []string{
			"UPDATE refresh_tokens SET revoked_at = @at WHERE user_id = @user AND revoked_at IS NULL",
			"UPDATE password_resets SET used_at = @at WHERE user_id = @user AND used_at IS NULL",
			pendingEmailChangesUsed,
		}
		for _, statement := range statements {
			if err := tx.Exec(statement, map[string]any{"at": at, "user": user.ID, "keep": keep}).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.putUserSessions(ctx, ended...)

	return nil
}

type Grant struct {
	Application model.Application
	Scopes      []string
	FirstIssued time.Time
	LastIssued  time.Time
}

func (s *Store) UserGrants(ctx context.Context, user uuid.UUID, now time.Time) ([]Grant, error) {
	var tokens []model.RefreshToken
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", user, now).
		Order("created_at DESC").
		Find(&tokens).Error
	if err != nil {
		return nil, err
	}

	byApp := map[uuid.UUID]*Grant{}
	order := []uuid.UUID{}
	for _, token := range tokens {
		grant, seen := byApp[token.ApplicationID]
		if !seen {
			grant = &Grant{FirstIssued: token.AuthenticatedAt, LastIssued: token.CreatedAt}
			byApp[token.ApplicationID] = grant
			order = append(order, token.ApplicationID)
		}

		grant.Scopes = append(grant.Scopes, token.Scope)
		if token.AuthenticatedAt.Before(grant.FirstIssued) {
			grant.FirstIssued = token.AuthenticatedAt
		}
	}

	if len(order) == 0 {
		return []Grant{}, nil
	}

	var apps []model.Application
	if err := s.db.WithContext(ctx).Where("id IN ?", order).Find(&apps).Error; err != nil {
		return nil, err
	}
	found := make(map[uuid.UUID]model.Application, len(apps))
	for _, app := range apps {
		found[app.ID] = app
	}

	out := make([]Grant, 0, len(order))
	for _, id := range order {
		app, ok := found[id]
		if !ok {
			continue
		}

		grant := byApp[id]
		grant.Application = app
		out = append(out, *grant)
	}

	return out, nil
}

func (s *Store) RevokeUserGrant(ctx context.Context, user, application uuid.UUID, at time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND application_id = ? AND revoked_at IS NULL", user, application).
		Update("revoked_at", at)

	return result.RowsAffected, result.Error
}

const keyLock = brand.SigningKeysLock

// AddSigningKeyIfDue stores `key` unless its algorithm already has an unretired
// key newer than `dueBefore`. An advisory lock makes one of several servers
// create it; it reports whether this one did.
func (s *Store) AddSigningKeyIfDue(ctx context.Context, key *model.SigningKey, dueBefore time.Time) (bool, error) {
	added := false

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", keyLock).Error; err != nil {
			return err
		}

		var fresh int64
		err := tx.Model(&model.SigningKey{}).
			Where("algorithm = ? AND retired_at IS NULL AND created_at > ?", key.Algorithm, dueBefore).
			Count(&fresh).Error
		if err != nil || fresh > 0 {
			return err
		}

		added = true
		return translate(tx.Create(key).Error)
	})

	return added, err
}

func (s *Store) RetireSigningKeys(ctx context.Context, algorithm string, before, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.SigningKey{}).
		Where("algorithm = ? AND retired_at IS NULL AND created_at < ?", algorithm, before).
		Update("retired_at", at).Error
}

func (s *Store) DeleteSigningKeys(ctx context.Context, before time.Time) error {
	return s.db.WithContext(ctx).
		Where("retired_at IS NOT NULL AND retired_at < ?", before).
		Delete(&model.SigningKey{}).Error
}

// DeleteSigningKeysExcept removes every key but these: what revoking a
// compromised key means, since a published key is one APIs still trust.
func (s *Store) DeleteSigningKeysExcept(ctx context.Context, keep []string) error {
	return s.db.WithContext(ctx).Where("kid NOT IN ?", keep).Delete(&model.SigningKey{}).Error
}

func (s *Store) CreateEmailVerification(ctx context.Context, verification *model.EmailVerification) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(verification).Error)
}

func (s *Store) EmailVerificationByHash(ctx context.Context, hash string) (*model.EmailVerification, error) {
	var verification model.EmailVerification
	if err := s.db.WithContext(ctx).First(&verification, "token_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &verification, nil
}

func (s *Store) VerifyEmail(ctx context.Context, verification *model.EmailVerification, at time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.EmailVerification{}).
			Where("id = ? AND used_at IS NULL", verification.ID).
			Update("used_at", at)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAlreadyUsed
		}

		err := tx.Model(&model.EmailVerification{}).
			Where("user_id = ? AND used_at IS NULL", verification.UserID).
			Update("used_at", at).Error
		if err != nil {
			return err
		}

		changes := map[string]any{"is_email_verified": true}
		if verification.IsChange() {
			changes["email"] = model.NormalizeEmail(verification.NewEmail)
		}

		err = tx.Model(&model.User{}).Where("id = ?", verification.UserID).Updates(changes).Error

		return translate(err)
	})
}
