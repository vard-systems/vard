package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/cache"
	"vard/internal/model"
)

type SessionQuery struct {
	// Search is the start of an address — "ada" finds ada@example.com — so it
	// can use the prefix index on users.email rather than read every user.
	Search string
	UserID uuid.UUID
	After  uuid.UUID
	Limit  int
}

type ActiveSession struct {
	model.UserSession
	Email     string
	FirstName string
	LastName  string
}

func (s *Store) ActiveSessions(ctx context.Context, q SessionQuery, now time.Time) ([]ActiveSession, error) {
	query := s.db.WithContext(ctx).
		Table("user_sessions").
		Select("user_sessions.*, users.email, users.first_name, users.last_name").
		Joins("JOIN users ON users.id = user_sessions.user_id").
		Where("user_sessions.revoked_at IS NULL AND user_sessions.expires_at > ?", now)

	if search := model.NormalizeEmail(q.Search); search != "" {
		query = query.Where("users.email LIKE ?", likeEscaper.Replace(search)+"%")
	}
	if q.UserID != uuid.Nil {
		query = query.Where("user_sessions.user_id = ?", q.UserID)
	}
	if q.After != uuid.Nil {
		query = query.Where(
			"(user_sessions.created_at, user_sessions.id) < (SELECT created_at, id FROM user_sessions WHERE id = ?)",
			q.After,
		)
	}

	var sessions []ActiveSession
	err := query.
		Order("user_sessions.created_at DESC, user_sessions.id DESC").
		Limit(q.Limit).
		Scan(&sessions).Error

	return sessions, err
}

func (s *Store) UserSession(ctx context.Context, id uuid.UUID) (*model.UserSession, error) {
	var session model.UserSession
	if err := s.db.WithContext(ctx).First(&session, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &session, nil
}

func (s *Store) SignOutUser(ctx context.Context, user uuid.UUID, at time.Time) (sessions, tokens int64, err error) {
	var ended endedSessions

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if ended, err = revokeUserSessionsIn(tx, at, "user_id = ? AND expires_at > ?", user, at); err != nil {
			return err
		}
		sessions = int64(len(ended.sessions))

		result := tx.Model(&model.RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", user, at).
			Update("revoked_at", at)
		if result.Error != nil {
			return result.Error
		}
		tokens = result.RowsAffected

		return tx.Exec(pendingEmailChangesUsed, map[string]any{"at": at, "user": user}).Error
	})
	if err == nil {
		s.sessionsEnded(ctx, ended)
	}

	return sessions, tokens, err
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
	return s.revokeUserSessions(ctx, at, "id = ?", id)
}

func (s *Store) RevokeUserSessionsFor(ctx context.Context, user uuid.UUID, at time.Time) error {
	return s.revokeUserSessions(ctx, at, "user_id = ?", user)
}

func (s *Store) revokeUserSessions(ctx context.Context, at time.Time, condition string, args ...any) error {
	var ended endedSessions
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		ended, err = revokeUserSessionsIn(tx, at, condition, args...)
		return err
	})
	if err == nil {
		s.sessionsEnded(ctx, ended)
	}

	return err
}

// RememberSessionApplication records that an application was given an ID
// token in a session, so it is told when the session ends.
func (s *Store) RememberSessionApplication(ctx context.Context, session, application uuid.UUID) error {
	link := model.SessionApplication{SessionID: session, ApplicationID: application}

	return s.db.WithContext(ctx).
		Omit(clause.Associations).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&link).Error
}

// LogoutNotice is an application to tell, over the back channel, that a
// session it signed someone in through has ended.
type LogoutNotice struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
	Email     string

	ApplicationID uuid.UUID
	Application   string
	ClientID      string
	URI           string
}

// OnLogout sets what is called, after the commit, with the applications to
// tell whenever sessions are ended. The store only finds them; telling them
// is the provider's job.
func (s *Store) OnLogout(notify func(context.Context, []LogoutNotice)) {
	s.onLogout = notify
}

// endedSessions is what a revocation ended, and who has to hear of it. The
// applications are found in the revoking transaction, before a deleted user
// takes the record of them along.
type endedSessions struct {
	sessions []model.UserSession
	notices  []LogoutNotice
}

func (s *Store) sessionsEnded(ctx context.Context, ended endedSessions) {
	s.putUserSessions(ctx, ended.sessions...)

	if s.onLogout != nil && len(ended.notices) > 0 {
		s.onLogout(ctx, ended.notices)
	}
}

func (s *Store) ActiveUserSessions(ctx context.Context, user uuid.UUID, now time.Time) ([]model.UserSession, error) {
	var sessions []model.UserSession
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", user, now).
		Order("authenticated_at DESC").
		Find(&sessions).Error

	return sessions, err
}

type userSession struct {
	ID              uuid.UUID  `json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	UserID          uuid.UUID  `json:"user_id"`
	AuthenticatedAt time.Time  `json:"authenticated_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	RevokedAt       *time.Time `json:"revoked_at"`
	IP              string     `json:"ip"`
	UserAgent       string     `json:"user_agent"`
}

func newUserSession(session model.UserSession) userSession {
	return userSession{
		ID:              session.ID,
		CreatedAt:       session.CreatedAt,
		UserID:          session.UserID,
		AuthenticatedAt: session.AuthenticatedAt,
		ExpiresAt:       session.ExpiresAt,
		RevokedAt:       session.RevokedAt,
		IP:              session.IP,
		UserAgent:       session.UserAgent,
	}
}

func (u userSession) model(hash string) model.UserSession {
	session := model.UserSession{
		TokenHash:       hash,
		UserID:          u.UserID,
		AuthenticatedAt: u.AuthenticatedAt,
		ExpiresAt:       u.ExpiresAt,
		RevokedAt:       u.RevokedAt,
		IP:              u.IP,
		UserAgent:       u.UserAgent,

		ID:        u.ID,
		CreatedAt: u.CreatedAt}

	return session
}

func revokeUserSessionsIn(tx *gorm.DB, at time.Time, condition string, args ...any) (endedSessions, error) {
	var ended endedSessions
	err := tx.Model(&ended.sessions).
		Clauses(clause.Returning{}).
		Where("revoked_at IS NULL").
		Where(condition, args...).
		Update("revoked_at", at).Error
	if err != nil || len(ended.sessions) == 0 {
		return ended, err
	}

	ids := make([]uuid.UUID, len(ended.sessions))
	for i, session := range ended.sessions {
		ids[i] = session.ID
	}

	err = tx.Raw(`
		SELECT links.session_id, sessions.user_id, users.email,
			apps.id AS application_id, apps.name AS application, apps.client_id,
			apps.backchannel_logout_uri AS uri
		FROM session_applications links
		JOIN user_sessions sessions ON sessions.id = links.session_id
		JOIN users ON users.id = sessions.user_id
		JOIN applications apps ON apps.id = links.application_id
		WHERE links.session_id IN ? AND sessions.expires_at > ?
			AND apps.backchannel_logout_uri <> '' AND apps.is_enabled`, ids, at).
		Scan(&ended.notices).Error

	return ended, err
}

func (s *Store) putUserSessions(ctx context.Context, sessions ...model.UserSession) {
	ctx, cancel := afterCommit(ctx)
	defer cancel()

	for _, session := range sessions {
		s.sessions.PutSession(ctx, cache.UserSession, session.TokenHash, newUserSession(session), session.ExpiresAt)
	}
}

// FrontchannelTarget is an application to load in the browser's frame when a
// session it signed someone in through ends there.
type FrontchannelTarget struct {
	ClientID string
	URI      string
}

// FrontchannelTargets are the applications of a session that are told of a
// logout in the browser.
func (s *Store) FrontchannelTargets(ctx context.Context, session uuid.UUID) ([]FrontchannelTarget, error) {
	var targets []FrontchannelTarget
	err := s.db.WithContext(ctx).Raw(`
		SELECT apps.client_id, apps.frontchannel_logout_uri AS uri
		FROM session_applications links
		JOIN applications apps ON apps.id = links.application_id
		WHERE links.session_id = ? AND apps.frontchannel_logout_uri <> '' AND apps.is_enabled
		ORDER BY apps.name`, session).
		Scan(&targets).Error

	return targets, err
}

func (s *Store) CreateFrontchannelLogout(ctx context.Context, logout *model.FrontchannelLogout) error {
	return s.db.WithContext(ctx).Create(logout).Error
}

// ClaimFrontchannelLogout takes a logout's page, once: ErrNotFound when there
// is none, ErrAlreadyUsed when it was shown or has expired.
func (s *Store) ClaimFrontchannelLogout(ctx context.Context, hash string, at time.Time) (*model.FrontchannelLogout, error) {
	var logout model.FrontchannelLogout
	if err := s.db.WithContext(ctx).First(&logout, "handle_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	result := s.db.WithContext(ctx).Model(&model.FrontchannelLogout{}).
		Where("id = ? AND used_at IS NULL AND expires_at > ?", logout.ID, at).
		Update("used_at", at)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrAlreadyUsed
	}

	return &logout, nil
}

// UseDPoPProof records a DPoP proof as used, once: false when it was already.
func (s *Store) UseDPoPProof(ctx context.Context, hash string, expires time.Time) (bool, error) {
	proof := model.DPoPProof{Hash: hash, ExpiresAt: expires}
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&proof)

	return result.RowsAffected == 1, result.Error
}
