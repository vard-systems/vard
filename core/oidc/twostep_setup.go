package oidc

import (
	"context"
	"encoding/base64"
	"errors"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"vard/core/brand"
	"vard/core/mail"
	"vard/core/model"
	"vard/core/store"
	"vard/tools/recovery"
	"vard/tools/totp"
)

// Two-step verification from the account page: what the user has, adding an
// authenticator or a passkey, removing one, and new recovery codes.

func (s *Service) TwoStep(ctx context.Context, session *Session) (*TwoStepStatus, error) {
	offered, err := s.twoStepOffered(ctx)
	if err != nil {
		return nil, err
	}

	factors, err := s.confirmedFactors(ctx, session.User.ID)
	if err != nil {
		return nil, err
	}

	left, err := s.store.RecoveryCodesLeft(ctx, session.User.ID)
	if err != nil {
		return nil, err
	}

	status := &TwoStepStatus{
		Offered:           offered && session.InBrowser(),
		Enabled:           len(factors) > 0,
		AsksPassword:      session.User.PasswordHash != "",
		Passkeys:          []FactorInfo{},
		RecoveryCodesLeft: left,
	}
	for _, factor := range factors {
		info := FactorInfo{ID: factor.ID.String(), Label: factor.Label, AddedAt: *factor.ConfirmedAt, LastUsedAt: factor.LastUsedAt}
		if isAuthenticator(factor) {
			status.Authenticator = &info
		} else {
			status.Passkeys = append(status.Passkeys, info)
		}
	}

	return status, nil
}

// factorOwner is whose factor is being set up, and where: on the account
// page, by a session whose password was just typed again, or while signing
// in to an application that requires one, by the waiting sign-in its httpOnly
// cookie names, whose first factor was proved minutes ago.
type factorOwner struct {
	user   *model.User
	signIn *model.PendingSignIn
}

// during is where the factor was added, for the activity log.
func (o factorOwner) during() string {
	if o.signIn != nil {
		return "sign_in"
	}
	return "account"
}

// BeginAuthenticator starts setting up an authenticator app: a fresh seed,
// not used until a code from it is typed back. Adding a way to sign in asks
// for the password, as taking one away does: otherwise a session left open,
// or stolen, could add a factor of its own and keep a way in.
func (s *Service) BeginAuthenticator(ctx context.Context, session *Session, password string, client Client) (*AuthenticatorSetup, error) {
	if err := s.provePresence(ctx, session, password, client); err != nil {
		return nil, err
	}

	factors, err := s.addingFactor(ctx, session)
	if err != nil {
		return nil, err
	}

	return s.startAuthenticator(ctx, factorOwner{user: session.User}, factors)
}

// ConfirmAuthenticator finishes setting up the authenticator with a code from
// it. When it is the user's first factor the answer carries their recovery
// codes, the only time they are shown.
func (s *Service) ConfirmAuthenticator(ctx context.Context, session *Session, code string, client Client) ([]string, error) {
	if _, err := s.addingFactor(ctx, session); err != nil {
		return nil, err
	}

	owner := factorOwner{user: session.User}
	factor, err := s.checkAuthenticator(ctx, owner, code)
	if err != nil {
		return nil, err
	}

	return s.confirmFactor(ctx, owner, factor, client)
}

// BeginPasskey starts adding a passkey: the request for
// navigator.credentials.create, kept until the browser answers. A passkey
// signs in on its own, so adding one asks for the password first.
func (s *Service) BeginPasskey(ctx context.Context, session *Session, password string, client Client) (*protocol.PublicKeyCredentialCreationOptions, error) {
	if err := s.provePresence(ctx, session, password, client); err != nil {
		return nil, err
	}

	factors, err := s.addingFactor(ctx, session)
	if err != nil {
		return nil, err
	}

	return s.startPasskey(ctx, factorOwner{user: session.User}, factors)
}

// FinishPasskey keeps the passkey the browser made, under the name the user
// gave it.
func (s *Service) FinishPasskey(ctx context.Context, session *Session, label string, response []byte, client Client) ([]string, error) {
	if _, err := s.addingFactor(ctx, session); err != nil {
		return nil, err
	}

	owner := factorOwner{user: session.User}
	factor, err := s.checkNewPasskey(ctx, owner, label, response)
	if err != nil {
		return nil, err
	}

	return s.confirmFactor(ctx, owner, factor, client)
}

// startAuthenticator keeps a fresh seed for the owner, unconfirmed.
func (s *Service) startAuthenticator(ctx context.Context, owner factorOwner, factors []model.UserFactor) (*AuthenticatorSetup, error) {
	if slices.ContainsFunc(factors, isAuthenticator) {
		return nil, ErrAuthenticatorExists
	}

	seed, err := totp.NewSecret()
	if err != nil {
		return nil, err
	}
	// The id is the factor's before it is stored, so the seed is sealed to it.
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sealed, err := s.seal(id, []byte(seed))
	if err != nil {
		return nil, err
	}

	factor := model.UserFactor{UserID: owner.user.ID, Method: model.MFAMethodTOTP, Secret: sealed}
	factor.ID = id
	if err := s.store.StartUserFactor(ctx, &factor); err != nil {
		return nil, err
	}

	organization := brand.Name
	if found, err := s.store.Organization(ctx); err == nil && found.Name != "" {
		organization = found.Name
	}

	return &AuthenticatorSetup{Secret: seed, URI: totp.URI(organization, owner.user.Email, seed)}, nil
}

// checkAuthenticator is the owner's unconfirmed authenticator, once a code
// from it is right; nothing is confirmed yet.
func (s *Service) checkAuthenticator(ctx context.Context, owner factorOwner, code string) (*model.UserFactor, error) {
	factors, err := s.store.UserFactors(ctx, owner.user.ID)
	if err != nil {
		return nil, err
	}
	index := slices.IndexFunc(factors, func(f model.UserFactor) bool {
		return f.Method == model.MFAMethodTOTP && !f.IsConfirmed()
	})
	if index < 0 {
		return nil, ErrSecondFactorExpired
	}
	factor := factors[index]

	seed, err := s.unseal(factor.ID, factor.Secret)
	if err != nil {
		return nil, err
	}

	step, ok := totp.Verify(string(seed), code, s.now(), 0)
	if !ok {
		return nil, ErrSecondFactorInvalid
	}
	used := time.Unix(step*int64(totp.Period/time.Second), 0)
	factor.LastUsedAt = &used

	return &factor, nil
}

// startPasskey keeps a registration ceremony for the owner, unconfirmed.
func (s *Service) startPasskey(ctx context.Context, owner factorOwner, factors []model.UserFactor) (*protocol.PublicKeyCredentialCreationOptions, error) {
	credentials, err := s.passkeyCredentials(factors)
	if err != nil {
		return nil, err
	}
	if len(credentials) >= maxPasskeys {
		return nil, ErrPasskeyLimit
	}

	exclusions := make([]protocol.CredentialDescriptor, len(credentials))
	for i, credential := range credentials {
		exclusions[i] = credential.Descriptor()
	}

	creation, ceremony, err := s.webauthn.BeginRegistration(passkeyUser{owner.user, credentials},
		webauthn.WithExclusions(exclusions),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementPreferred))
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sealed, err := s.sealJSON(id, ceremony)
	if err != nil {
		return nil, err
	}

	factor := model.UserFactor{UserID: owner.user.ID, Method: model.MFAMethodWebAuthn, Secret: sealed}
	factor.ID = id
	if err := s.store.StartUserFactor(ctx, &factor); err != nil {
		return nil, err
	}

	return &creation.Response, nil
}

// checkNewPasskey is the owner's unconfirmed passkey holding the credential the
// browser made, once it answers the ceremony; nothing is confirmed yet.
func (s *Service) checkNewPasskey(ctx context.Context, owner factorOwner, label string, response []byte) (*model.UserFactor, error) {
	factors, err := s.store.UserFactors(ctx, owner.user.ID)
	if err != nil {
		return nil, err
	}
	index := slices.IndexFunc(factors, func(f model.UserFactor) bool {
		return f.Method == model.MFAMethodWebAuthn && !f.IsConfirmed()
	})
	if index < 0 {
		return nil, ErrSecondFactorExpired
	}
	factor := factors[index]

	var ceremony webauthn.SessionData
	if err := s.unsealJSON(factor.ID, factor.Secret, &ceremony); err != nil {
		return nil, err
	}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return nil, ErrPasskeyRefused
	}

	credential, err := s.webauthn.CreateCredential(passkeyUser{user: owner.user}, ceremony, parsed)
	if err != nil {
		return nil, ErrPasskeyRefused
	}

	if factor.Secret, err = s.sealJSON(factor.ID, credential); err != nil {
		return nil, err
	}
	factor.CredentialID = base64.RawURLEncoding.EncodeToString(credential.ID)
	factor.Label = label

	return &factor, nil
}

// RemoveFactor removes one of the user's factors. Taking protection away
// asks for the password, when the account has one, as moving the address
// does: an unattended session is not enough.
func (s *Service) RemoveFactor(ctx context.Context, session *Session, id, password string, client Client) error {
	if err := s.provePresence(ctx, session, password, client); err != nil {
		return err
	}

	factor, err := uuid.Parse(id)
	if err != nil {
		return ErrNotYours
	}

	removed, err := s.store.RemoveUserFactor(ctx, session.User.ID, factor)
	if err != nil {
		return err
	}
	if !removed {
		return ErrNotYours
	}

	s.record(ctx, session.User, session.User.Email, "user.two_step_removed", client, nil)

	return nil
}

// NewRecoveryCodes replaces the user's recovery codes; the old ones stop
// working.
func (s *Service) NewRecoveryCodes(ctx context.Context, session *Session, password string, client Client) ([]string, error) {
	if err := s.provePresence(ctx, session, password, client); err != nil {
		return nil, err
	}

	has, err := s.store.HasUserFactor(ctx, session.User.ID)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrTwoStepNotOffered
	}

	codes, err := s.replaceRecoveryCodes(ctx, session.User)
	if err != nil {
		return nil, err
	}

	s.record(ctx, session.User, session.User.Email, "user.recovery_codes_regenerated", client, nil)

	return codes, nil
}

// confirmFactor makes a factor count, tells the account it was added, and
// hands out recovery codes with the user's first.
func (s *Service) confirmFactor(ctx context.Context, owner factorOwner, factor *model.UserFactor, client Client) ([]string, error) {
	user := owner.user
	had, err := s.store.HasUserFactor(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// Set up while signing in, it is the account's first or nothing: of two
	// sign-ins setting one up at once, only one keeps its factor.
	err = s.store.ConfirmUserFactor(ctx, factor, s.now(), owner.signIn != nil)
	switch {
	case errors.Is(err, store.ErrAlreadyUsed):
		return nil, ErrSecondFactorExpired
	case errors.Is(err, store.ErrHasFactor):
		return nil, ErrTwoStepNotOffered
	case err != nil:
		return nil, err
	}

	method := MethodAuthenticator
	if factor.Method == model.MFAMethodWebAuthn {
		method = MethodPasskey
	}
	s.record(ctx, user, user.Email, "user.two_step_enabled", client, map[string]any{"method": method, "during": owner.during()})
	s.mailFactorAdded(ctx, user, method, client)

	if had {
		return nil, nil
	}

	return s.replaceRecoveryCodes(ctx, user)
}

func (s *Service) replaceRecoveryCodes(ctx context.Context, user *model.User) ([]string, error) {
	codes, hashes, err := recovery.New(s.sealer, userRecoveryPurpose)
	if err != nil {
		return nil, err
	}

	return codes, s.store.ReplaceRecoveryCodes(ctx, user.ID, hashes)
}

// addingFactor checks a factor may be added here, and returns the user's
// factors so far.
func (s *Service) addingFactor(ctx context.Context, session *Session) ([]model.UserFactor, error) {
	if !session.InBrowser() {
		return nil, ErrBrowserRequired
	}

	offered, err := s.twoStepOffered(ctx)
	if err != nil {
		return nil, err
	}
	if !offered {
		return nil, ErrTwoStepNotOffered
	}

	return s.confirmedFactors(ctx, session.User.ID)
}

// provePresence asks the person at the keyboard for the account's password,
// when it has one. In a browser only: an application's token never changes
// how its user signs in.
func (s *Service) provePresence(ctx context.Context, session *Session, password string, client Client) error {
	if !session.InBrowser() {
		return ErrBrowserRequired
	}

	// An account with no password (one that signs in with another account)
	// has nothing to type: a sign-in of the last few minutes stands in.
	if session.User.PasswordHash == "" {
		if s.now().Sub(session.Record.AuthenticatedAt) > presenceWindow {
			return ErrSignInAgain
		}
		return nil
	}

	return s.checkCurrentPassword(ctx, session.User, password, client)
}

// presenceWindow is how recent a sign-in has to be to stand in for a
// password, for an account that has none.
const presenceWindow = 10 * time.Minute

// ErrSignInAgain asks a person without a password to sign in again before a
// change to how they sign in: an old session is not enough.
var ErrSignInAgain = problem("sign_in_again")

// twoStepOffered is whether the account page offers two-step verification:
// some flow in use has its step. The default flow is always in use; another
// only while it is on, as only then do its applications sign in through it.
func (s *Service) twoStepOffered(ctx context.Context) (bool, error) {
	flow, err := s.flowFor(ctx, "")
	if err != nil {
		return false, err
	}
	if flow.Offers(model.StepTOTP) {
		return true, nil
	}

	return s.store.EnabledLoginFlowOffers(ctx, model.StepTOTP)
}

// mailFactorAdded tells the account a second factor was added to it, from the
// account page or a sign-in alike: whoever adds one is asked for it at every
// sign-in after, so its owner hears of one they did not add. The factor is
// already kept, so a failure to send is only logged.
func (s *Service) mailFactorAdded(ctx context.Context, user *model.User, method string, client Client) {
	text := s.textIn(ctx, client.Language)

	params := map[string]any{
		"email":  user.Email,
		"method": text.Localize("email.two_step_on."+method, nil),
		"ip":     client.IP,
	}

	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := s.mail.Send(sendCtx, mail.Message{
		To:      user.Email,
		Subject: text.Localize("email.two_step_on.subject", params),
		Body:    text.Localize("email.two_step_on.body", params),
	})
	if err != nil {
		s.log.Warn("sending the two-step notice failed", "user", user.ID, "error", err)
	}
}
