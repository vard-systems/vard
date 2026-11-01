package store

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"vard/core/model"
)

// WriteAudit adds a line to the activity log, first fitting outside input to
// its column so an oversized or invalid User-Agent cannot keep an event out of
// the log.
func (s *Store) WriteAudit(ctx context.Context, entry *model.AuditLog) error {
	entry.UserAgent = model.Truncate(entry.UserAgent, 255)
	entry.ActorEmail = model.Truncate(entry.ActorEmail, 255)
	entry.IP = model.Truncate(entry.IP, 45)

	return s.db.WithContext(ctx).Create(entry).Error
}

func (s *Store) AuditLog(ctx context.Context, limit int) ([]model.AuditLog, error) {
	events, _, err := s.AuditQuery(ctx, AuditFilter{Limit: limit})
	return events, err
}

type AuditCursor struct {
	At time.Time
	ID uuid.UUID
}

type AuditFilter struct {
	Search         string
	Actions        []string
	Actor          string
	From, To       time.Time
	Before         *AuditCursor
	HideUserActors bool
	Limit          int
}

func (s *Store) AuditQuery(ctx context.Context, filter AuditFilter) ([]model.AuditLog, *AuditCursor, error) {
	query := s.db.WithContext(ctx).Model(&model.AuditLog{})

	const byUser = "(admin_id IS NULL AND target_type = 'user')"

	if term := strings.TrimSpace(filter.Search); term != "" {
		like := "%" + likeEscaper.Replace(term) + "%"
		actor := "actor_email ILIKE @like"
		if filter.HideUserActors {
			actor = "(actor_email ILIKE @like AND NOT " + byUser + ")"
		}

		query = query.Where(
			actor+" OR action ILIKE @like OR ip ILIKE @like OR target_id ILIKE @like",
			map[string]any{"like": like},
		)
	}

	if len(filter.Actions) > 0 {
		query = query.Where("action IN ?", filter.Actions)
	}

	if filter.Actor != "" {
		query = query.Where("actor_email = ?", filter.Actor)
		if filter.HideUserActors {
			query = query.Where("NOT " + byUser)
		}
	}

	if !filter.From.IsZero() {
		query = query.Where("created_at >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		query = query.Where("created_at < ?", filter.To)
	}

	if filter.Before != nil {
		query = query.Where("(created_at, id) < (?, ?)", filter.Before.At, filter.Before.ID)
	}

	var events []model.AuditLog
	err := query.Order("created_at DESC, id DESC").Limit(filter.Limit + 1).Find(&events).Error
	if err != nil {
		return nil, nil, err
	}

	if len(events) <= filter.Limit {
		return events, nil, nil
	}

	events = events[:filter.Limit]
	last := events[len(events)-1]

	return events, &AuditCursor{At: last.CreatedAt, ID: last.ID}, nil
}

const (
	actionLogin            = "admin.login"
	actionLoginFailed      = "admin.login_failed"
	actionLoginBlocked     = "admin.login_blocked"
	actionUserLogin        = "user.login"
	actionUserLoginFailed  = "user.login_failed"
	actionUserLoginBlocked = "user.login_blocked"
)

// Counts are the numbers on the dashboard: how much of everything there is.
type Counts struct {
	Users               int64 `json:"users"`
	ActiveUsers         int64 `json:"active_users"`
	NewUsers            int64 `json:"new_users"`
	Applications        int64 `json:"applications"`
	EnabledApplications int64 `json:"enabled_applications"`
	APIs                int64 `json:"apis"`
	APIScopes           int64 `json:"api_scopes"`
	UserRoles           int64 `json:"user_roles"`
	Admins              int64 `json:"admins"`
	LockedAdmins        int64 `json:"locked_admins"`
	ActiveSessions      int64 `json:"active_sessions"`
}

func (s *Store) Counts(ctx context.Context, now, since time.Time) (Counts, error) {
	var counts Counts

	err := s.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM users) AS users,
			(SELECT COUNT(*) FROM users WHERE is_active) AS active_users,
			(SELECT COUNT(*) FROM users WHERE created_at >= @since) AS new_users,
			(SELECT COUNT(*) FROM applications) AS applications,
			(SELECT COUNT(*) FROM applications WHERE is_enabled) AS enabled_applications,
			(SELECT COUNT(*) FROM apis) AS apis,
			(SELECT COUNT(*) FROM api_scopes) AS api_scopes,
			(SELECT COUNT(*) FROM user_roles) AS user_roles,
			(SELECT COUNT(*) FROM admins) AS admins,
			(SELECT COUNT(*) FROM sign_in_throttles
				WHERE key LIKE 'admin:%' AND blocked_until > @now) AS locked_admins,
			(SELECT COUNT(*) FROM admin_sessions
				WHERE revoked_at IS NULL AND expires_at > @now) AS active_sessions`,
		map[string]any{"now": now, "since": since},
	).Scan(&counts).Error

	return counts, err
}

// SignIns is how signing in to the panel has gone since a moment.
type SignIns struct {
	Since     time.Time `json:"since"`
	Succeeded int64     `json:"succeeded"`
	Failed    int64     `json:"failed"`
	Blocked   int64     `json:"blocked"`
}

func (s *Store) SignInsSince(ctx context.Context, since time.Time) (SignIns, error) {
	out := SignIns{Since: since}

	err := s.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) FILTER (WHERE action IN (@login, @user_login)) AS succeeded,
			COUNT(*) FILTER (WHERE action IN (@failed, @user_failed)) AS failed,
			COUNT(*) FILTER (WHERE action IN (@blocked, @user_blocked)) AS blocked
		FROM audit_logs
		WHERE created_at >= @since AND action IN (
			@login, @failed, @blocked, @user_login, @user_failed, @user_blocked
		)`,
		map[string]any{
			"since": since, "login": actionLogin, "failed": actionLoginFailed, "blocked": actionLoginBlocked,
			"user_login": actionUserLogin, "user_failed": actionUserLoginFailed, "user_blocked": actionUserLoginBlocked,
		},
	).Scan(&out).Error

	return out, err
}

// DayCount is one day of the activity chart.
type DayCount struct {
	// Day is the date in the database's time zone, as YYYY-MM-DD.
	Day    string `json:"day"`
	Events int64  `json:"events"`
	// Failures are the sign-ins refused that day, wrong passwords and
	// blocked accounts together.
	Failures int64 `json:"failures"`
}

func (s *Store) DailyActivity(ctx context.Context, now time.Time, days int) ([]DayCount, error) {
	out := []DayCount{}

	err := s.db.WithContext(ctx).Raw(`
		WITH activity AS (
			SELECT
				date_trunc('day', created_at) AS day,
				COUNT(*) AS events,
				COUNT(*) FILTER (WHERE action IN (@failed, @blocked, @user_failed, @user_blocked)) AS failures
			FROM audit_logs
			WHERE created_at >= date_trunc('day', CAST(@now AS timestamptz)) - make_interval(days => @back)
				AND created_at < date_trunc('day', CAST(@now AS timestamptz)) + interval '1 day'
			GROUP BY date_trunc('day', created_at)
		)
		SELECT
			to_char(d.day, 'YYYY-MM-DD') AS day,
			COALESCE(a.events, 0) AS events,
			COALESCE(a.failures, 0) AS failures
		FROM generate_series(
			date_trunc('day', CAST(@now AS timestamptz)) - make_interval(days => @back),
			date_trunc('day', CAST(@now AS timestamptz)),
			interval '1 day'
		) AS d(day)
		LEFT JOIN activity a ON a.day = d.day
		ORDER BY d.day`,
		map[string]any{
			"now": now, "back": days - 1, "failed": actionLoginFailed, "blocked": actionLoginBlocked,
			"user_failed": actionUserLoginFailed, "user_blocked": actionUserLoginBlocked,
		},
	).Scan(&out).Error

	return out, err
}

// ActorCount is how much one administrator did.
type ActorCount struct {
	Actor  string `json:"actor"`
	Events int64  `json:"events"`
}

func (s *Store) TopActors(ctx context.Context, since time.Time, limit int) ([]ActorCount, error) {
	// Empty rather than nil: a week with no changes is an empty list in the
	// JSON, which the panel can count, and not null, which it cannot.
	out := []ActorCount{}

	err := s.db.WithContext(ctx).Raw(`
		SELECT actor_email AS actor, COUNT(*) AS events
		FROM audit_logs
		WHERE created_at >= @since AND actor_email <> '' AND action NOT LIKE 'admin.log%'
			AND admin_id IS NOT NULL
		GROUP BY actor_email
		ORDER BY events DESC, actor_email
		LIMIT @limit`,
		map[string]any{"since": since, "limit": limit},
	).Scan(&out).Error

	return out, err
}

type TargetName struct {
	Name          string
	ApplicationID *uuid.UUID
}

// targetTables whitelists where each target kind is stored, so nothing from the
// log reaches query text.
var targetTables = map[string]struct{ table, name, application string }{
	"user":            {table: "users", name: "email"},
	"user_field":      {table: "user_fields", name: "name"},
	"user_role":       {table: "user_roles", name: "name", application: "application_id"},
	"application":     {table: "applications", name: "name", application: "id"},
	"api":             {table: "apis", name: "name"},
	"admin_user":      {table: "admins", name: "username"},
	"admin_role":      {table: "admin_roles", name: "name"},
	"login_flow":      {table: "login_flows", name: "name"},
	"language":        {table: "languages", name: "name"},
	"sso_connection":  {table: "sso_connections", name: "name"},
	"social_provider": {table: "social_providers", name: "name"},
}

func (s *Store) TargetNames(ctx context.Context, targetType string, ids []string) (map[string]TargetName, error) {
	where, ok := targetTables[targetType]
	names := map[string]TargetName{}

	valid := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if parsed, err := uuid.Parse(id); err == nil {
			valid = append(valid, parsed)
		}
	}

	if !ok || len(valid) == 0 {
		return names, nil
	}

	application := "NULL::uuid"
	if where.application != "" {
		application = where.application
	}

	var rows []struct {
		ID            uuid.UUID
		Name          string
		ApplicationID *uuid.UUID
	}

	err := s.db.WithContext(ctx).
		Table(where.table).
		Select("id, "+where.name+" AS name, "+application+" AS application_id").
		Where("id IN ?", valid).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		names[row.ID.String()] = TargetName{Name: row.Name, ApplicationID: row.ApplicationID}
	}

	return names, nil
}
