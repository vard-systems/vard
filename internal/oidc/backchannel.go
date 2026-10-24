package oidc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"loginer/internal/jose"
	"loginer/internal/model"
	"loginer/internal/store"
)

// OpenID Connect Back-Channel Logout 1.0.
const (
	backchannelLogoutEvent = "http://schemas.openid.net/event/backchannel-logout"
	logoutTokenType        = "logout+jwt"

	// logoutTokenLifetime is short: the token is posted at once and an
	// application only has to check it then.
	logoutTokenLifetime = 2 * time.Minute

	backchannelTimeout = 5 * time.Second

	// undelivered is the activity log's word for a logout no try got through.
	undelivered = "application.logout_undelivered"
	// backchannelSenders caps the posts under way at once, so signing one
	// user out of many sessions does not open a connection for each.
	backchannelSenders = 8
)

// backchannelRetries are the waits before each try after the first. Only an
// answer that may change is tried again: no answer, or the application's own
// failure (5xx). A 4xx is the application refusing the token, which another
// try would not change.
var backchannelRetries = []time.Duration{2 * time.Second, 8 * time.Second}

// backchannelLogout tells applications, server to server, that sessions they
// signed someone in through have ended. Posts run after the response; one that
// never gets through is written to the activity log, where administrators see
// which application missed a logout.
type backchannelLogout struct {
	client  *http.Client
	running sync.WaitGroup

	// stopping is closed when the server shuts down: retries still waiting
	// give up rather than hold the shutdown.
	stopping chan struct{}
	stopped  sync.Once
}

// errRefused is an answer another try would not change.
type errRefused struct{ status int }

func (e errRefused) Error() string { return fmt.Sprintf("the application answered %d", e.status) }

// newBackchannelClient goes through the same address checks as every other
// outbound request to an administrator's URL, and follows no redirect: a
// logout token is for the URI the application registered and nowhere else.
func newBackchannelClient(allowLoopback bool) *http.Client {
	client := newFederationClient(backchannelTimeout, allowLoopback)
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return client
}

// TellApplications posts a logout token to each application to be told. It
// returns at once; Wait waits for the posts under way.
func (s *Service) TellApplications(ctx context.Context, notices []store.LogoutNotice) {
	if len(notices) == 0 {
		return
	}

	// The request that ended the sessions may be over before the posts are.
	ctx = context.WithoutCancel(ctx)

	s.backchannel.running.Go(func() {
		senders := make(chan struct{}, backchannelSenders)
		var sending sync.WaitGroup

		for _, notice := range notices {
			senders <- struct{}{}
			sending.Go(func() {
				defer func() { <-senders }()

				s.deliverLogout(ctx, notice)
			})
		}

		sending.Wait()
	})
}

// Wait waits for the logout posts under way, until ctx ends. Retries still
// waiting are given up.
func (s *Service) Wait(ctx context.Context) {
	s.backchannel.stopped.Do(func() { close(s.backchannel.stopping) })

	done := make(chan struct{})
	go func() {
		s.backchannel.running.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}

// deliverLogout tries until the logout gets through, the application refuses
// it, or the tries run out, and records it in the activity log if it never got
// through.
func (s *Service) deliverLogout(ctx context.Context, notice store.LogoutNotice) {
	err := s.sendLogout(ctx, notice)
	tries := 1

	for _, wait := range backchannelRetries {
		if err == nil || errors.As(err, new(errRefused)) {
			break
		}

		select {
		case <-time.After(wait):
		case <-s.backchannel.stopping:
			wait = -1
		}
		if wait < 0 {
			break
		}

		err = s.sendLogout(ctx, notice)
		tries++
	}

	if err == nil {
		return
	}

	s.log.Warn("telling an application of a logout failed",
		"client_id", notice.ClientID, "session", notice.SessionID, "tries", tries, "error", err)

	entry := model.AuditLog{
		ActorEmail: notice.Email,
		Action:     undelivered,
		TargetType: "application",
		TargetID:   notice.ApplicationID.String(),
		Metadata: map[string]any{
			"application": notice.Application, "user": notice.UserID.String(),
			"tries": tries, "reason": err.Error(),
		},
	}
	if err := s.store.WriteAudit(ctx, &entry); err != nil {
		s.log.Error("writing the activity log failed", "error", err, "action", undelivered)
	}
}

func (s *Service) sendLogout(ctx context.Context, notice store.LogoutNotice) error {
	token, err := s.logoutToken(notice)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, backchannelTimeout)
	defer cancel()

	body := url.Values{"logout_token": {token}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, notice.URI, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.backchannel.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))

	// §2.8: 200 says the logout succeeded; 204 is common enough to accept.
	switch {
	case resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent:
		return nil
	case resp.StatusCode >= 500:
		return fmt.Errorf("the application answered %d", resp.StatusCode)
	default:
		return errRefused{resp.StatusCode}
	}

}

// logoutToken is the JWT of §2.4: who and which session, for which
// application, with the logout event and never a nonce, signed like an ID
// token but typed so it cannot be taken for one.
func (s *Service) logoutToken(notice store.LogoutNotice) (string, error) {
	jti, err := uuid.NewV7()
	if err != nil {
		return "", err
	}

	key, ok := s.keys.signing(idTokenAlgorithm)
	if !ok {
		return "", errors.New("oidc: no signing key for " + idTokenAlgorithm)
	}

	now := s.now()

	return jose.Sign(key, logoutTokenType, map[string]any{
		"iss":    s.issuer,
		"sub":    notice.UserID.String(),
		"aud":    notice.ClientID,
		"iat":    now.Unix(),
		"exp":    now.Add(logoutTokenLifetime).Unix(),
		"jti":    jti.String(),
		"sid":    notice.SessionID.String(),
		"events": map[string]any{backchannelLogoutEvent: map[string]any{}},
	})
}
