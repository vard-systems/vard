package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"loginer/internal/mail"
	"loginer/internal/model"
	"loginer/internal/store"
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

func (s *Service) VerifyEmail(ctx context.Context, token string, client Client) error {
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

	err = s.store.VerifyEmail(ctx, verification, s.now())
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
