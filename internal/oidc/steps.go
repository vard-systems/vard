package oidc

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"

	"loginer/internal/model"
	"loginer/internal/store"
)

// The Agreements and Consent steps run after the user is known and before a
// code is issued: finish sends the browser to their page while one still has
// to be passed, and the page's answer comes back here and finishes again.

const (
	PageAgreements = "/agreements"
	PageConsent    = "/consent"
)

// Answers to the client when prompt=none meets a page the user has to see
// (OpenID Connect Core §3.1.2.6).
const (
	ErrConsentRequired     = "consent_required"
	ErrInteractionRequired = "interaction_required"
)

// ErrBrowserRequired refuses an answer to the Agreements or Consent page that
// came with an application's access token: only the person, in their own
// browser, may accept terms or allow an application.
var ErrBrowserRequired = problem("browser_session_required")

// stepBefore is the page this sign-in still has to pass before its code, or
// "" when there is none.
func (s *Service) stepBefore(ctx context.Context, req *model.AuthorizationRequest, user *model.User) (string, error) {
	flow, err := s.store.EffectiveLoginFlow(ctx, req.Application)
	if err != nil {
		return "", err
	}

	if flow.Offers(model.StepTerms) {
		organization, err := s.store.Organization(ctx)
		if err != nil {
			return "", err
		}
		if !organization.AgreementsAccepted(user.AgreementsAcceptedAt) {
			return PageAgreements, nil
		}
	}

	if flow.Offers(model.StepConsent) {
		if req.ForceConsent {
			return PageConsent, nil
		}

		consent, err := s.store.Consent(ctx, user.ID, req.ApplicationID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			return PageConsent, nil
		case err != nil:
			return "", err
		case !consent.Covers(strings.Fields(req.Scope)):
			return PageConsent, nil
		}
	}

	return "", nil
}

// silentRefusal is what prompt=none answers when the user would have to see
// a page: the client is told, and nothing is shown.
func silentRefusal(page string) (code, description string) {
	if page == PageConsent {
		return ErrConsentRequired, "the user has not allowed this application what it asks for"
	}

	return ErrInteractionRequired, "the user has to accept the terms before signing in"
}

// RequestedScope is one thing an application asks for, as the consent page
// lists it. Description is the API's own words for an API scope; standard
// scopes are described by the page in the reader's language.
type RequestedScope struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	API         string `json:"api,omitempty"`
}

type ConsentPage struct {
	Application PublicApplication `json:"application"`
	Scopes      []RequestedScope  `json:"scopes"`
}

// ConsentFor is what the consent page shows for a sign-in under way.
func (s *Service) ConsentFor(ctx context.Context, handle string, session *Session) (*ConsentPage, error) {
	if !session.InBrowser() {
		return nil, ErrBrowserRequired
	}

	req, err := s.pending(ctx, handle)
	if err != nil {
		return nil, err
	}

	return s.describeRequest(ctx, req.Application, req.Scope, req.Audience)
}

// describeRequest is what an application asks for, as a page lists it: the
// standard scopes by name, an API's scopes in the API's own words.
func (s *Service) describeRequest(ctx context.Context, app *model.Application, scope, audience string) (*ConsentPage, error) {
	page := &ConsentPage{Application: NewPublicApplication(app), Scopes: []RequestedScope{}}

	var api *model.API
	if audience != "" {
		found, err := s.store.APIByIdentifier(ctx, audience)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
		api = found
	}

	for _, name := range strings.Fields(scope) {
		requested := RequestedScope{Name: name}
		if api != nil && !slices.Contains(model.Scopes, name) {
			requested.API = api.Name
			if index := slices.IndexFunc(api.Scopes, func(one model.APIScope) bool { return one.Name == name }); index >= 0 {
				requested.Description = api.Scopes[index].Description
			}
		}
		page.Scopes = append(page.Scopes, requested)
	}

	return page, nil
}

// AnswerConsent records the user's answer on the consent page and says where
// the browser goes: on with the sign-in, or back to the application refused.
func (s *Service) AnswerConsent(ctx context.Context, handle string, session *Session, allow bool, client Client) (string, error) {
	if !session.InBrowser() {
		return "", ErrBrowserRequired
	}

	req, err := s.pending(ctx, handle)
	if err != nil {
		return "", err
	}

	details := map[string]any{"application": req.Application.Name, "scope": req.Scope}

	if !allow {
		s.record(ctx, session.User, session.User.Email, "user.consent_denied", client, details)
		return s.refuse(ctx, req, "the user did not allow the application")
	}

	if err := s.store.AllowScopes(ctx, session.User.ID, req.ApplicationID, strings.Fields(req.Scope)); err != nil {
		return "", err
	}
	if req.ForceConsent {
		if err := s.store.ConsentAsked(ctx, req); err != nil {
			return "", err
		}
	}

	s.record(ctx, session.User, session.User.Email, "user.consent_given", client, details)

	return s.finish(ctx, req, handle, session.User, &session.Record)
}

// AnswerAgreements records whether the user accepted the organization's
// terms and privacy policy, and says where the browser goes.
func (s *Service) AnswerAgreements(ctx context.Context, handle string, session *Session, accept bool, client Client) (string, error) {
	if !session.InBrowser() {
		return "", ErrBrowserRequired
	}

	req, err := s.pending(ctx, handle)
	if err != nil {
		return "", err
	}

	if !accept {
		s.record(ctx, session.User, session.User.Email, "user.agreements_declined", client, nil)
		return s.refuse(ctx, req, "the user did not accept the terms")
	}

	if err := s.store.AcceptAgreements(ctx, session.User, s.now()); err != nil {
		return "", err
	}

	s.record(ctx, session.User, session.User.Email, "user.agreements_accepted", client, nil)

	return s.finish(ctx, req, handle, session.User, &session.Record)
}

// refuse ends a sign-in the user said no to, so the same request cannot be
// answered again, and sends the browser back to the application with
// access_denied.
func (s *Service) refuse(ctx context.Context, req *model.AuthorizationRequest, description string) (string, error) {
	if err := s.store.CompleteAuthorizationRequest(ctx, req, s.now()); errors.Is(err, store.ErrAlreadyUsed) {
		return "", ErrRequestExpired
	} else if err != nil {
		return "", err
	}

	return s.redirectError(req.RedirectURI, req.State, ErrAccessDenied, description), nil
}

func (s *Service) stepPage(page, handle string) string {
	return withQuery(s.accountURL+page, url.Values{"request": {handle}})
}
