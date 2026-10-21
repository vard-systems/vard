package oidc

import (
	"context"
	"errors"
	"net/url"
	"slices"

	"loginer/internal/jose"
	"loginer/internal/model"
	"loginer/internal/store"
)

type LogoutParams struct {
	IDTokenHint           string
	ClientID              string
	PostLogoutRedirectURI string
	State                 string
	// Confirmed is set by the confirmation page's own call, never read from
	// the logout request: a hint-less logout signs out only once the user
	// has said yes.
	Confirmed bool
}

// Logout ends this browser's session and returns where to send it: the
// application's registered post-logout URI, or the signed-out page.
//
// A request without id_token_hint, from a browser that is signed in, signs
// nobody out: it goes to the confirmation page instead (RP-Initiated Logout
// §2), since nothing says the application sent it.
//
// `signedOut` is false when the request was refused or its id_token_hint names
// someone else. The cookie is cleared only when true, because this is a GET
// anyone can put in a link. Refresh tokens are left alone: each application
// ends its own session.
func (s *Service) Logout(ctx context.Context, p LogoutParams, sessionToken string, client Client) (location string, signedOut bool, err error) {
	clientID := p.ClientID

	var subject string

	if p.IDTokenHint != "" {
		// The hint names who asked; it may well have expired, which is fine.
		// Its signature and issuer still have to be this server's.
		var claims struct {
			Issuer   string `json:"iss"`
			Subject  string `json:"sub"`
			Audience any    `json:"aud"`
		}
		if _, err := jose.Verify(p.IDTokenHint, s.keys.lookup(ctx), &claims); err != nil || claims.Issuer != s.issuer {
			return s.errorPage(ErrInvalidRequest, "id_token_hint is not an ID token from this server"), false, nil
		}

		audience, _ := claims.Audience.(string)
		if clientID != "" && clientID != audience {
			return s.errorPage(ErrInvalidRequest, "client_id does not match the id_token_hint"), false, nil
		}
		clientID = audience
		subject = claims.Subject
	}

	var app *model.Application
	if clientID != "" {
		found, err := s.store.ApplicationByClientID(ctx, clientID)
		switch {
		case errors.Is(err, store.ErrNotFound):
			return s.errorPage(ErrInvalidRequest, "no application has this client_id"), false, nil
		case err != nil:
			return "", false, err
		}
		app = found
	}

	// Only a URI the application registered is followed, so a logout link
	// cannot be used to send someone anywhere.
	if p.PostLogoutRedirectURI != "" {
		if app == nil {
			return s.errorPage(ErrInvalidRequest, "post_logout_redirect_uri needs id_token_hint or client_id"), false, nil
		}
		if !slices.Contains(app.PostLogoutRedirectURIs, p.PostLogoutRedirectURI) {
			return s.errorPage(ErrInvalidRequest, "post_logout_redirect_uri is not registered for this application"), false, nil
		}
	}

	if p.IDTokenHint == "" && !p.Confirmed {
		session, err := s.SessionFor(ctx, sessionToken)
		if err != nil {
			return "", false, err
		}
		if session != nil {
			values := url.Values{"post_logout_redirect_uri": {p.PostLogoutRedirectURI}, "state": {p.State}}
			if app != nil {
				values.Set("client_id", app.ClientID)
			}
			return withQuery(s.accountURL+PageLogout, values), false, nil
		}
	}

	signedOut, err = s.endSession(ctx, sessionToken, subject, client)
	if err != nil {
		return "", false, err
	}

	if p.PostLogoutRedirectURI != "" {
		return withQuery(p.PostLogoutRedirectURI, url.Values{"state": {p.State}}), signedOut, nil
	}

	values := url.Values{}
	if app != nil {
		values.Set("client_id", app.ClientID)
	}

	return withQuery(s.accountURL+PageLoggedOut, values), signedOut, nil
}

func (s *Service) endSession(ctx context.Context, sessionToken, subject string, client Client) (bool, error) {
	if subject != "" {
		session, err := s.SessionFor(ctx, sessionToken)
		if err != nil {
			return false, err
		}
		if session != nil && session.User.ID.String() != subject {
			s.log.Info("a logout named another user than the one signed in here",
				"hint", subject, "session", session.User.ID)
			return false, nil
		}
	}

	return true, s.SignOut(ctx, sessionToken, client)
}
