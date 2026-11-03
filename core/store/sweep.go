package store

import (
	"context"
	"log/slog"
	"time"

	"vard/core/model"
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
		n, err := s.deleteInBatches(ctx, table, "expires_at < ?", now)
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
		n, err := s.deleteInBatches(ctx, &model.AuditLog{}, "created_at < ?", auditBefore)
		removed += n
		if err != nil {
			return removed, err
		}
	}

	return removed, nil
}

func (s *Store) deleteInBatches(ctx context.Context, table any, where string, arg any) (int64, error) {
	var removed int64

	for {
		batch := s.db.Model(table).Select("id").Where(where, arg).Limit(sweepBatch)

		result := s.db.WithContext(ctx).Where("id IN (?)", batch).Delete(table)
		if result.Error != nil {
			return removed, translate(result.Error)
		}

		removed += result.RowsAffected
		if result.RowsAffected < sweepBatch {
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
