package store

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vard/internal/model"
)

// An unverified account may have been registered by somebody other than the
// address's owner. When the owner first proves the address, whatever such a
// stranger could have left behind as a way back in goes: linked identities,
// factors, recovery codes, consents, waiting sign-ins, codes and devices,
// sessions and refresh tokens.
var addressProofStatements = []string{
	"DELETE FROM social_identities WHERE user_id = @user",
	"DELETE FROM sso_identities WHERE user_id = @user",
	"DELETE FROM user_factors WHERE user_id = @user",
	"DELETE FROM user_recovery_codes WHERE user_id = @user",
	"DELETE FROM consents WHERE user_id = @user",
	"DELETE FROM pending_sign_ins WHERE user_id = @user",
	"UPDATE login_codes SET used_at = @at WHERE user_id = @user AND used_at IS NULL",
	"UPDATE device_authorizations SET status = @denied WHERE user_id = @user AND used_at IS NULL",
}

// firstProofIn locks the user's row and says whether its address is still
// unproved, so two proofs racing clear it once.
func firstProofIn(tx *gorm.DB, user uuid.UUID) (bool, error) {
	var verified []bool
	err := tx.Raw("SELECT is_email_verified FROM users WHERE id = ? FOR UPDATE", user).Scan(&verified).Error
	if err != nil || len(verified) == 0 {
		return false, err
	}

	return !verified[0], nil
}

// clearBeforeProofIn ends what the account held before its address was
// proved, and records it in the activity log within the same transaction.
func clearBeforeProofIn(tx *gorm.DB, user uuid.UUID, at time.Time, entry model.AuditLog) (endedSessions, error) {
	ended, err := revokeUserSessionsIn(tx, at, "user_id = ?", user)
	if err != nil {
		return ended, err
	}
	if _, err := revokeUserRefreshIn(tx, at, user); err != nil {
		return ended, err
	}

	for _, statement := range addressProofStatements {
		if err := tx.Exec(statement, map[string]any{"at": at, "user": user, "denied": model.DeviceDenied}).Error; err != nil {
			return ended, err
		}
	}

	entry.Action = "user.address_proved"
	entry.TargetType = "user"
	entry.TargetID = user.String()
	entry.UserAgent = model.Truncate(entry.UserAgent, 255)
	entry.ActorEmail = model.Truncate(entry.ActorEmail, 255)
	entry.IP = model.Truncate(entry.IP, 45)

	return ended, tx.Create(&entry).Error
}
