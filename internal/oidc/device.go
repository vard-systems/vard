package oidc

import (
	"context"
	"encoding/hex"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"vard/internal/model"
	"vard/internal/store"
)

// The device authorization grant (RFC 8628).

const (
	PathDeviceAuthorization = "/oauth2/device_authorization"
	// PageDevice is where the person enters the code.
	PageDevice = "/device"

	userCodePurpose = "device user code"
)

// The answers a polling device gets while it waits (RFC 8628 §3.5).
const (
	ErrAuthorizationPending = "authorization_pending"
	ErrSlowDown             = "slow_down"
	ErrExpiredToken         = "expired_token"
)

// ErrDeviceCodeInvalid is a user code that names no device waiting: mistyped,
// expired or already answered. One answer for all, so it says nothing about
// which codes exist.
var ErrDeviceCodeInvalid = problem("device_code_invalid")

// ErrDeviceRefused is a device the person allowed but cannot use: the
// application's rules give their account no token. The device is refused.
var ErrDeviceRefused = problem("device_refused")

// DeviceStart is the answer to a device asking to sign someone in.
type DeviceStart struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// StartDevice begins a device's sign-in.
func (s *Service) StartDevice(ctx context.Context, client ClientAuth, scope, audience string) (*DeviceStart, error) {
	app, err := s.authenticate(ctx, client)
	if err != nil {
		return nil, err
	}

	switch {
	case !app.IsEnabled:
		return nil, oauthError(ErrUnauthorizedClient, "the application is disabled")
	case !slices.Contains(app.GrantTypes, model.GrantDeviceCode):
		return nil, oauthError(ErrUnauthorizedClient, "the application does not have the device code grant")
	case strings.TrimSpace(scope) == "":
		return nil, oauthError(ErrInvalidScope, "scope is required")
	}

	if audience != "" {
		if _, err := s.store.APIByIdentifier(ctx, audience); errors.Is(err, store.ErrNotFound) {
			return nil, errUnknownAudience
		} else if err != nil {
			return nil, err
		}
	}

	deviceCode, deviceHash, err := model.NewSecret()
	if err != nil {
		return nil, err
	}

	// A code already waiting is drawn again; with this many, rarely more than once.
	for range 5 {
		userCode, err := model.NewUserCode()
		if err != nil {
			return nil, err
		}

		device := model.DeviceAuthorization{
			DeviceCodeHash: deviceHash,
			UserCodeHash:   s.userCodeHash(userCode),
			ApplicationID:  app.ID,
			Scope:          strings.Join(strings.Fields(scope), " "),
			Audience:       audience,
			Status:         model.DevicePending,
			Interval:       model.DeviceInterval,
			ExpiresAt:      s.now().Add(model.DeviceAuthorizationLifetime),
		}
		err = s.store.CreateDeviceAuthorization(ctx, &device)
		if errors.Is(err, store.ErrDuplicate) {
			continue
		}
		if err != nil {
			return nil, err
		}

		verify := s.accountURL + PageDevice
		return &DeviceStart{
			DeviceCode:              deviceCode,
			UserCode:                userCode,
			VerificationURI:         verify,
			VerificationURIComplete: withQuery(verify, url.Values{"user_code": {userCode}}),
			ExpiresIn:               int(model.DeviceAuthorizationLifetime.Seconds()),
			Interval:                model.DeviceInterval,
		}, nil
	}

	return nil, errors.New("oidc: no free user code after five tries")
}

// deviceToken is the device asking whether the person has answered yet.
func (s *Service) deviceToken(ctx context.Context, app *model.Application, p TokenParams) (*TokenResponse, error) {
	if p.DeviceCode == "" {
		return nil, oauthError(ErrInvalidRequest, "device_code is required")
	}

	device, err := s.store.DeviceByCode(ctx, model.HashSecret(p.DeviceCode))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, oauthError(ErrInvalidGrant, "the device code is not valid")
	case err != nil:
		return nil, err
	case device.ApplicationID != app.ID:
		return nil, oauthError(ErrInvalidGrant, "the device code was issued to another client")
	}

	now := s.now()

	switch {
	case device.UsedAt != nil:
		return nil, oauthError(ErrInvalidGrant, "the device code has already been used")
	case device.Status == model.DeviceDenied:
		return nil, oauthError(ErrAccessDenied, "the user did not allow the device")
	case !now.Before(device.ExpiresAt):
		return nil, oauthError(ErrExpiredToken, "the device code has expired; start again")
	case device.Status == model.DevicePending:
		tooSoon := device.LastPolledAt != nil && now.Sub(*device.LastPolledAt) < time.Duration(device.Interval)*time.Second
		if err := s.store.PolledDevice(ctx, device.ID, now, tooSoon); err != nil {
			return nil, err
		}
		if tooSoon {
			return nil, oauthError(ErrSlowDown, "ask less often")
		}
		return nil, oauthError(ErrAuthorizationPending, "the user has not answered yet")
	}

	claimed, err := s.store.ClaimDevice(ctx, device.ID, now)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, oauthError(ErrInvalidGrant, "the device code has already been used")
	}

	user, err := s.store.User(ctx, *device.UserID)
	if errors.Is(err, store.ErrNotFound) || err == nil && !user.CanSignIn() {
		return nil, oauthError(ErrInvalidGrant, "the user can no longer sign in")
	}
	if err != nil {
		return nil, err
	}

	response, err := s.issue(ctx, grant{
		app:      app,
		user:     user,
		scopes:   strings.Fields(device.Scope),
		audience: device.Audience,
		authTime: *device.AuthenticatedAt,
		methods:  device.Methods,
		session:  device.SessionID,
		dpopKey:  p.dpopKey,
		refusal:  ErrInvalidGrant,
	})
	if err != nil {
		return nil, err
	}

	if device.SessionID != nil && response.IDToken != "" {
		if err := s.store.RememberSessionApplication(ctx, *device.SessionID, app.ID); err != nil {
			s.log.Error("recording the application a session signed in failed", "error", err)
		}
	}

	return response, nil
}

// DeviceFor is what the person sees before allowing a device: the
// application and what it asks for.
func (s *Service) DeviceFor(ctx context.Context, userCode string, session *Session) (*ConsentPage, error) {
	if !session.InBrowser() {
		return nil, ErrBrowserRequired
	}

	device, err := s.waitingDevice(ctx, userCode)
	if err != nil {
		return nil, err
	}

	return s.describeRequest(ctx, device.Application, device.Scope, device.Audience)
}

// AnswerDevice allows or refuses a device for the signed-in person. Allowing
// it is checked as a sign-in would be, so a user who could not get a token is
// refused here rather than by the device's next ask.
func (s *Service) AnswerDevice(ctx context.Context, userCode string, session *Session, allow bool, client Client) error {
	if !session.InBrowser() {
		return ErrBrowserRequired
	}

	device, err := s.waitingDevice(ctx, userCode)
	if err != nil {
		return err
	}

	refused := false
	if allow {
		token, err := s.evaluate(ctx, device.Application, session.User, strings.Fields(device.Scope), device.Audience)
		if err != nil {
			return err
		}
		refused = !token.Issued
		allow = token.Issued
	}

	answered, err := s.store.AnswerDevice(ctx, device, allow, session.User.ID, &session.Record, s.now())
	if err != nil {
		return err
	}
	if !answered {
		return ErrDeviceCodeInvalid
	}

	action := "user.device_denied"
	if allow {
		action = "user.device_allowed"
	}
	s.record(ctx, session.User, session.User.Email, action, client, map[string]any{"application": device.Application.Name})

	if refused {
		return ErrDeviceRefused
	}

	return nil
}

func (s *Service) waitingDevice(ctx context.Context, userCode string) (*model.DeviceAuthorization, error) {
	device, err := s.store.DeviceByUserCode(ctx, s.userCodeHash(userCode), s.now())
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrDeviceCodeInvalid
	case err != nil:
		return nil, err
	case device.Application == nil || !device.Application.IsEnabled:
		return nil, ErrDeviceCodeInvalid
	}

	return device, nil
}

// userCodeHash keys the code with the server's secret: eight letters are far
// too few for a bare hash to keep.
func (s *Service) userCodeHash(code string) string {
	return hex.EncodeToString(s.sealer.Tag(userCodePurpose, []byte(model.NormalizeUserCode(code))))
}
