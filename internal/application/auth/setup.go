package auth

import (
	"context"
	"time"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
)

// SetupPage is where the console sets up its first administrator.
const SetupPage = "/new-super-admin"

// IssueSetupToken makes the one-time token setting the panel up takes, while
// no administrator exists, and returns it to be logged; "" once one does.
// Only its hash is kept.
func IssueSetupToken(ctx context.Context, st *store.Store, now time.Time) (string, error) {
	token, hash, err := model.NewSecret()
	if err != nil {
		return "", err
	}

	issued, err := st.IssueSetupToken(ctx, &model.SetupToken{TokenHash: hash, ExpiresAt: now.Add(model.SetupTokenLifetime)})
	if err != nil || !issued {
		return "", err
	}

	return token, nil
}
