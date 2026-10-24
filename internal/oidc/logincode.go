package oidc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"loginer/internal/mail"
	"loginer/internal/model"
	"loginer/internal/store"
)

// The emailed code step: after the password is accepted, a code is sent to the
// account's address and the session starts only when it is typed back. Sign-ins
// whose provider already proved the address skip it.
//
// The handle is the real secret: a six-digit code is guessable, so it only
// works in the browser holding the handle.

// CodeChallenge is a sign-in waiting for its emailed code, as the page shows
// it.
type CodeChallenge struct {
	// Handle names the sign-in. The page keeps it and sends it back.
	Handle string `json:"handle"`

	// Email is masked: enough to recognise one's own address, not to learn
	// someone else's.
	Email string `json:"email"`

	ExpiresAt time.Time `json:"expires_at"`

	// ResendAfter is seconds until another code may be sent, for a countdown.
	ResendAfter int `json:"resend_after"`

	// AttemptsLeft is how many more codes may be typed before this sign-in
	// is spent and has to be started again.
	AttemptsLeft int `json:"attempts_left"`
}

type codeContext struct {
	code     *model.LoginCode
	settings *model.OTPSettings
}

func (s *Service) sendLoginCode(
	ctx context.Context,
	user *model.User,
	request string,
	remember bool,
	client Client,
) (*SignInResult, error) {
	settings, err := s.store.OTPSettings(ctx)
	if err != nil {
		return nil, err
	}

	handle, handleHash, err := model.NewSecret()
	if err != nil {
		return nil, err
	}

	code, codeHash, err := model.NewOTP(settings.CodeLength)
	if err != nil {
		return nil, err
	}

	now := s.now()
	waiting := model.LoginCode{
		HandleHash: handleHash,
		CodeHash:   codeHash,
		UserID:     user.ID,
		Request:    request,
		RememberMe: remember,
		SentAt:     now,
		ExpiresAt:  now.Add(settings.Lifetime()),
	}
	if err := s.store.CreateLoginCode(ctx, &waiting); err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, "user.login_code_sent", client, nil)

	s.mailCode(ctx, user, code, request, client, *settings)

	answer := challenge(handle, waiting, *settings, user.Email, now)

	return &SignInResult{Code: &answer}, nil
}

func (s *Service) SubmitLoginCode(ctx context.Context, handle, typed string, client Client) (*SignInResult, error) {
	waiting, err := s.waitingCode(ctx, handle)
	if err != nil {
		return nil, err
	}

	user := waiting.code.User
	if user == nil || !user.CanSignIn(s.now()) {
		return nil, ErrInvalidCredentials
	}

	// The guess is taken before the code is looked at, so of several typed
	// at once only as many as the sign-in has left are ever compared.
	attempts, ok, err := s.store.RecordLoginCodeAttempt(ctx, waiting.code, waiting.settings.MaxAttempts)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCodeAttemptsUsed
	}

	if !waiting.code.MatchesOTP(typed) {
		s.record(ctx, user, user.Email, "user.login_code_failed", client, map[string]any{
			"attempts": attempts, "of": waiting.settings.MaxAttempts,
		})

		if attempts >= waiting.settings.MaxAttempts {
			return nil, ErrCodeAttemptsUsed
		}

		return nil, ErrCodeInvalid
	}

	// The code is right, so it is spent — whether or not what follows works,
	// and whoever else is typing it at the same moment.
	used, err := s.store.ConsumeLoginCode(ctx, waiting.code, s.now())
	if err != nil {
		return nil, err
	}
	if !used {
		return nil, ErrCodeExpired
	}

	flow, err := s.flowFor(ctx, waiting.code.Request)
	if err != nil {
		return nil, err
	}

	return s.startSession(ctx, user, flow, waiting.code.Request, waiting.code.RememberMe, client, "user.login")
}

func (s *Service) ResendLoginCode(ctx context.Context, handle string, client Client) (*CodeChallenge, error) {
	waiting, err := s.waitingCode(ctx, handle)
	if err != nil {
		return nil, err
	}

	user := waiting.code.User
	if user == nil || !user.CanSignIn(s.now()) {
		return nil, ErrInvalidCredentials
	}

	now := s.now()
	if wait := waiting.code.SentAt.Add(waiting.settings.Resend()); now.Before(wait) {
		return nil, ErrCodeTooSoon
	}

	code, codeHash, err := model.NewOTP(waiting.settings.CodeLength)
	if err != nil {
		return nil, err
	}

	if err := s.store.ResendLoginCode(ctx, waiting.code, codeHash, now); err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, "user.login_code_sent", client, map[string]any{"resent": true})

	s.mailCode(ctx, user, code, waiting.code.Request, client, *waiting.settings)

	answer := challenge(handle, *waiting.code, *waiting.settings, user.Email, now)

	return &answer, nil
}

func (s *Service) waitingCode(ctx context.Context, handle string) (*codeContext, error) {
	if handle == "" {
		return nil, ErrCodeExpired
	}

	settings, err := s.store.OTPSettings(ctx)
	if err != nil {
		return nil, err
	}

	code, err := s.store.LoginCodeByHash(ctx, model.HashSecret(handle))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrCodeExpired
	case err != nil:
		return nil, err
	}

	if !code.Usable(s.now(), settings.MaxAttempts) {
		return nil, ErrCodeExpired
	}

	return &codeContext{code: code, settings: settings}, nil
}

func challenge(handle string, code model.LoginCode, settings model.OTPSettings, email string, now time.Time) CodeChallenge {
	wait := max(int(code.SentAt.Add(settings.Resend()).Sub(now).Round(time.Second).Seconds()), 0)

	return CodeChallenge{
		Handle:       handle,
		Email:        maskEmail(email),
		ExpiresAt:    code.ExpiresAt,
		ResendAfter:  wait,
		AttemptsLeft: max(settings.MaxAttempts-code.Attempts, 0),
	}
}

func (s *Service) mailCode(
	ctx context.Context,
	user *model.User,
	code, request string,
	client Client,
	settings model.OTPSettings,
) {
	text := s.textIn(ctx, client.Language)

	name := text.Localize("email.reset.your_account", nil)
	if pending, err := s.pending(ctx, request); err == nil {
		name = pending.Application.Name
	}

	params := map[string]any{
		"app":     name,
		"email":   user.Email,
		"code":    code,
		"minutes": settings.LifetimeMinutes,
	}

	msg := mail.Message{
		To:      user.Email,
		Subject: text.Localize("email.code.subject", params),
		Body:    text.Localize("email.code.body", params),
	}

	go func() {
		sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()

		if err := s.mail.Send(sendCtx, msg); err != nil {
			s.log.Error("sending a one-time code failed", "error", err)
		}
	}()
}

func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 1 {
		return email
	}

	local, domain := email[:at], email[at:]
	if len(local) <= 2 {
		return strings.Repeat("•", len(local)) + domain
	}

	return fmt.Sprintf("%c%s%c%s", local[0], strings.Repeat("•", len(local)-2), local[len(local)-1], domain)
}
