package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"vard/core/mail"
	"vard/core/model"
	"vard/core/store"
)

func (s *Service) sendVerification(ctx context.Context, user *model.User, request string, client Client) error {
	return s.mailVerification(ctx, user, "", request, client)
}

func (s *Service) mailVerification(
	ctx context.Context,
	user *model.User,
	newEmail, request string,
	client Client,
) error {
	token, hash, err := model.NewSecret()
	if err != nil {
		return err
	}

	err = s.store.CreateEmailVerification(ctx, &model.EmailVerification{
		TokenHash: hash,
		UserID:    user.ID,
		NewEmail:  newEmail,
		ExpiresAt: s.now().Add(model.EmailVerificationLifetime),
	})
	if err != nil {
		return err
	}

	text := s.textIn(ctx, client.Language)

	name := text.Localize("email.reset.your_account", nil)
	if pending, err := s.pending(ctx, request); err == nil {
		name = pending.Application.Name
	} else {
		request = ""
	}

	to := user.Email
	if newEmail != "" {
		to = newEmail
	}

	params := map[string]any{
		"app":   name,
		"email": to,
		"link":  withQuery(s.accountURL+PageVerify, url.Values{"token": {token}, "request": {request}}),
		"hours": int(model.EmailVerificationLifetime.Hours()),
	}

	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = s.mail.Send(sendCtx, mail.Message{
		To:      to,
		Subject: text.Localize("email.verify.subject", params),
		Body:    text.Localize("email.verify.body", params),
	})
	if err != nil {
		return fmt.Errorf("send the verification email: %w", err)
	}

	s.record(ctx, user, to, "user.email_verification_sent", client, nil)

	return nil
}

// ErrVerificationPasswordRequired asks for the account's password before a
// link that proves an address for it is taken, and ErrVerificationSignInRequired
// for a browser signed in to it, where the account has no password.
var (
	ErrVerificationPasswordRequired = problem("verification_password_required")
	ErrVerificationSignInRequired   = problem("verification_sign_in_required")
)

// VerifyEmail takes a verification link. Who follows it has to show they hold
// the account as well as the inbox — its password, or a browser signed in to
// it — whenever the link would hand them something: an address not proved
// yet, or a new address. Otherwise whoever registered somebody else's address,
// or changed their own account's to it, would have the owner confirm an
// account whose password the stranger chose, and sign in to it afterwards.
// password is "" when none was typed; session is the browser's, nil for none.
func (s *Service) VerifyEmail(ctx context.Context, token, password string, session *Session, client Client) error {
	if token == "" {
		return ErrVerificationInvalid
	}

	verification, err := s.store.EmailVerificationByHash(ctx, model.HashSecret(token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return ErrVerificationInvalid
	case err != nil:
		return err
	case !verification.Usable(s.now()):
		return ErrVerificationInvalid
	}

	// A change of address may not move an account off a domain that must use
	// SSO, even with an older link.
	if verification.IsChange() {
		owner, err := s.store.User(ctx, verification.UserID)
		if err != nil {
			return err
		}
		if err := s.ssoRequiredFor(ctx, owner.Email); err != nil {
			return err
		}
	}

	owner, err := s.store.User(ctx, verification.UserID)
	if err != nil {
		return err
	}

	// Checked before the link is spent, so a wrong password leaves it for
	// the right one; each guess is counted as a sign-in's is.
	if err := s.heldByTheirOwner(ctx, owner, verification, password, session, client); err != nil {
		return err
	}

	err = s.store.VerifyEmail(ctx, verification, s.now(), proof(owner.Email, client))
	switch {
	case errors.Is(err, store.ErrAlreadyUsed):
		return ErrVerificationInvalid
	case errors.Is(err, store.ErrDuplicate):
		return ErrEmailTaken
	case err != nil:
		return err
	}

	user, err := s.store.User(ctx, verification.UserID)
	if err != nil {
		return err
	}

	action := "user.email_verified"
	if verification.IsChange() {
		action = "user.email_changed"
	}
	s.record(ctx, user, user.Email, action, client, nil)

	return nil
}

// proof is the activity log entry for the first proof of an address, which
// the store writes in the transaction that clears what came before it.
func proof(email string, client Client) model.AuditLog {
	return model.AuditLog{ActorEmail: email, IP: client.IP, UserAgent: client.UserAgent}
}

// heldByTheirOwner says whether the person following a verification link holds
// the account it is for, when the link would give them anything.
func (s *Service) heldByTheirOwner(ctx context.Context, owner *model.User, verification *model.EmailVerification, password string, session *Session, client Client) error {
	switch {
	case session != nil && session.User.ID == owner.ID:
		return nil
	case !verification.IsChange() && owner.IsEmailVerified:
		// Proved already: the link proves nothing new and clears nothing.
		return nil
	case !verification.IsChange() && owner.PasswordHash == "":
		// No password a stranger could have chosen, and the first proof
		// ends every other way in (clearBeforeProofIn).
		return nil
	case owner.PasswordHash == "":
		return ErrVerificationSignInRequired
	case password == "":
		return ErrVerificationPasswordRequired
	}

	return s.checkCurrentPassword(ctx, owner, password, client)
}
