package oidc

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"vard/core/model"
)

// Who may take a verification link without typing a password, decided before
// any password is looked at: the account's own browser; anyone, when the link
// gives nothing (an address proved already) or a stranger could have chosen
// no password (a first proof of a password-less account, which ends every
// other way in). A password-less account's new address takes its own browser.
func TestAVerificationLinkTakesTheAccountsHolder(t *testing.T) {
	s := &Service{}
	ctx := context.Background()

	withPassword := &model.User{Base: model.Base{ID: uuid.New()}, PasswordHash: "hash"}
	withoutPassword := &model.User{Base: model.Base{ID: uuid.New()}}
	verified := &model.User{Base: model.Base{ID: uuid.New()}, PasswordHash: "hash", IsEmailVerified: true}
	elsewhere := &Session{User: &model.User{Base: model.Base{ID: uuid.New()}}}

	first := &model.EmailVerification{}
	change := &model.EmailVerification{NewEmail: "new@example.com"}

	cases := []struct {
		name    string
		owner   *model.User
		link    *model.EmailVerification
		session *Session
		want    error
	}{
		{"the account's own browser", withPassword, change, &Session{User: withPassword}, nil},
		{"a first proof of a password-less account", withoutPassword, first, nil, nil},
		{"an address already proved", verified, first, nil, nil},
		{"a first proof, with no password typed", withPassword, first, nil, ErrVerificationPasswordRequired},
		{"a new address, with no password typed", verified, change, nil, ErrVerificationPasswordRequired},
		{"a first proof, from another account's browser", withPassword, first, elsewhere, ErrVerificationPasswordRequired},
		{"a password-less account's new address, elsewhere", withoutPassword, change, nil, ErrVerificationSignInRequired},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := s.heldByTheirOwner(ctx, c.owner, c.link, "", c.session, Client{}); !errors.Is(err, c.want) {
				t.Errorf("heldByTheirOwner() = %v, want %v", err, c.want)
			}
		})
	}
}
