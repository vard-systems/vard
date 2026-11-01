package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
)

// Signing in with a passkey alone. The browser offers whichever passkey the
// person picks — the passkey names the user, so nothing is typed — and the
// device has to verify them (a fingerprint, a face, a PIN): possession and
// that check together are both factors, so no second one is asked. Offered
// where two-step verification is, since that is where passkeys are added.

// ErrPasskeySignInFailed covers every way a passkey's answer is refused, so
// it says nothing about which accounts or passkeys exist.
var ErrPasskeySignInFailed = problem("passkey_sign_in_failed")

// PasskeyOffer is what the sign-in page hands navigator.credentials.get, and
// the handle it sends back with the answer.
type PasskeyOffer struct {
	Handle  string                                      `json:"handle"`
	Passkey *protocol.PublicKeyCredentialRequestOptions `json:"passkey"`
}

func (s *Service) BeginPasskeySignIn(ctx context.Context, request string, remember bool) (*PasskeyOffer, error) {
	flow, err := s.flowFor(ctx, request)
	if err != nil {
		return nil, err
	}
	if !flow.Offers(model.StepTOTP) {
		return nil, ErrTwoStepNotOffered
	}

	assertion, ceremony, err := s.webauthn.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(ceremony)
	if err != nil {
		return nil, err
	}

	handle, hash, err := model.NewSecret()
	if err != nil {
		return nil, err
	}

	login := model.PasskeyLogin{
		HandleHash: hash,
		Request:    request,
		RememberMe: remember,
		Ceremony:   string(encoded),
		ExpiresAt:  s.now().Add(model.PasskeyLoginLifetime),
	}
	if err := s.store.CreatePasskeyLogin(ctx, &login); err != nil {
		return nil, err
	}

	return &PasskeyOffer{Handle: handle, Passkey: &assertion.Response}, nil
}

// FinishPasskeySignIn checks the browser's answer and starts the session.
func (s *Service) FinishPasskeySignIn(ctx context.Context, handle string, response []byte, client Client) (*SignInResult, error) {
	now := s.now()

	login, err := s.store.ClaimPasskeyLogin(ctx, model.HashSecret(handle), now)
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, store.ErrAlreadyUsed):
		return nil, ErrRequestExpired
	case err != nil:
		return nil, err
	}

	var ceremony webauthn.SessionData
	if err := json.Unmarshal([]byte(login.Ceremony), &ceremony); err != nil {
		return nil, err
	}

	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return nil, ErrPasskeySignInFailed
	}

	// The passkey names its user by the handle it was made with: the user's id.
	var user *model.User
	whose := func(_, userHandle []byte) (webauthn.User, error) {
		id, err := uuid.FromBytes(userHandle)
		if err != nil {
			return nil, err
		}
		found, err := s.store.User(ctx, id)
		if err != nil {
			return nil, err
		}
		factors, err := s.confirmedFactors(ctx, found.ID)
		if err != nil {
			return nil, err
		}
		credentials, err := s.passkeyCredentials(factors)
		if err != nil {
			return nil, err
		}

		user = found
		return passkeyUser{found, credentials}, nil
	}

	_, credential, err := s.webauthn.ValidatePasskeyLogin(whose, ceremony, parsed)
	if err != nil || user == nil {
		return nil, ErrPasskeySignInFailed
	}

	// A counter that went backwards means two copies of the key.
	if credential.Authenticator.CloneWarning {
		s.log.Warn("a passkey's counter went backwards; refusing it", "user", user.ID)
		return nil, ErrPasskeySignInFailed
	}

	if !user.CanSignIn() {
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "locked or inactive"})
		return nil, ErrPasskeySignInFailed
	}

	factor, err := s.store.UserFactorByCredential(ctx, user.ID, base64.RawURLEncoding.EncodeToString(credential.ID))
	if err != nil {
		return nil, ErrPasskeySignInFailed
	}
	sealed, err := s.sealJSON(factor.ID, credential)
	if err != nil {
		return nil, err
	}
	if err := s.store.UsedPasskey(ctx, factor.ID, sealed, now); err != nil {
		return nil, err
	}

	flow, err := s.flowFor(ctx, login.Request)
	if err != nil {
		return nil, err
	}
	if !flow.Offers(model.StepTOTP) {
		return nil, ErrTwoStepNotOffered
	}
	if err := s.admit(ctx, user, flow, login.Request, client); err != nil {
		return nil, err
	}

	// The ceremony requires the person verified on the device, not only
	// the device, and the passkey is one of the account's own factors.
	methods := model.WithSecondFactor([]string{model.AMRHardwareKey, model.AMRUserPresence}, "")
	return s.openSession(ctx, user, flow, login.Request, login.RememberMe, methods, client, "user.login")
}
