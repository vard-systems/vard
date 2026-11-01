package store

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/cache"
)

const sweepInterval = time.Hour

// sweepBatch keeps each DELETE small so its locks never hold sign-ins up for
// long.
const sweepBatch = 5000

// expiring are the records that end at `expires_at`. A refresh token shares its
// family's expiry, so reuse detection never loses a token it would have caught.
var expiring = []any{
	&model.AuthorizationRequest{},
	&model.PushedRequest{},
	&model.DeviceAuthorization{},
	&model.AuthorizationCode{},
	&model.RefreshToken{},
	&model.UserSession{},
	&model.PasswordReset{},
	&model.EmailVerification{},
	&model.LoginCode{},
	&model.PendingSignIn{},
	&model.PasskeyLogin{},
	&model.FrontchannelLogout{},
	&model.AdminSession{},
	&model.SocialLogin{},
	&model.SSOLogin{},
	&model.SAMLAssertion{},
	&model.DPoPProof{},
	&model.TokenRevocation{},
	&model.SignInThrottle{},
}

func (s *Store) Sweep(ctx context.Context, now, auditBefore time.Time) (int64, error) {
	var removed int64

	for _, table := range expiring {
		n, err := s.deleteInBatches(ctx, table, sessionKinds[tableName(s.db, table)], "expires_at < ?", now)
		removed += n
		if err != nil {
			return removed, err
		}
	}

	// KeepSwept clears these every grace; this is for a server that sweeps
	// only through here.
	if _, err := s.ForgetRefreshSuccessors(ctx, now); err != nil {
		return removed, err
	}

	if !auditBefore.IsZero() {
		n, err := s.deleteInBatches(ctx, &model.AuditLog{}, "", "created_at < ?", auditBefore)
		removed += n
		if err != nil {
			return removed, err
		}
	}

	return removed, nil
}

var sessionKinds = map[string]string{
	"user_sessions":  cache.UserSession,
	"admin_sessions": cache.AdminSession,
}

func tableName(db *gorm.DB, model any) string {
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(model); err != nil {
		return ""
	}
	return statement.Schema.Table
}

func (s *Store) deleteInBatches(ctx context.Context, table any, kind, where string, arg any) (int64, error) {
	var removed int64

	for {
		batch := s.db.Model(table).Select("id").Where(where, arg).Limit(sweepBatch)

		var (
			deleted int64
			err     error
		)
		if kind == "" {
			result := s.db.WithContext(ctx).Where("id IN (?)", batch).Delete(table)
			deleted, err = result.RowsAffected, result.Error
		} else {
			// The table name is the model's, never anything a request said.
			var hashes []string
			err = s.db.WithContext(ctx).
				Raw("DELETE FROM "+tableName(s.db, table)+" WHERE id IN (?) RETURNING token_hash", batch).
				Scan(&hashes).Error
			deleted = int64(len(hashes))
			s.sessions.DropSessions(ctx, kind, hashes...)
		}
		if err != nil {
			return removed, translate(err)
		}

		removed += deleted
		if deleted < sweepBatch {
			return removed, nil
		}
	}
}

// KeepSwept sweeps at startup and every sweepInterval until ctx ends, keeping
// the log for `retention` (zero keeps all). Concurrent sweeps on several
// servers are harmless.
func (s *Store) KeepSwept(ctx context.Context, retention time.Duration, log *slog.Logger) {
	sweep := func() {
		now := time.Now()

		var auditBefore time.Time
		if retention > 0 {
			auditBefore = now.Add(-retention)
		}

		removed, err := s.Sweep(ctx, now, auditBefore)
		switch {
		case err != nil && ctx.Err() == nil:
			log.Error("sweeping expired records failed", "error", err, "removed", removed)
		case removed > 0:
			log.Info("swept expired records", "removed", removed)
		}
	}

	sweep()

	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	// A replaced refresh token's sealed successor is a copy of a live token,
	// kept only for a retry within the grace, so it goes within one more.
	successors := time.NewTicker(model.RefreshReuseGrace)
	defer successors.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		case <-successors.C:
			if _, err := s.ForgetRefreshSuccessors(ctx, time.Now()); err != nil && ctx.Err() == nil {
				log.Error("clearing sealed refresh tokens failed", "error", err)
			}
		}
	}
}
