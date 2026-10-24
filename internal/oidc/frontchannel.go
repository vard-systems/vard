package oidc

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"loginer/internal/model"
	"loginer/internal/store"
)

// OpenID Connect Front-Channel Logout 1.0. An application that ends its
// session in the browser gives a frontchannel_logout_uri; when a browser
// signs out, a page of this server loads each one of the session's in a
// hidden frame, with iss and sid, then carries on to where the logout was
// going. The page is the API's own, so its Content Security Policy can name
// exactly the applications it frames.

// PathFrontchannelLogout is the page that loads the applications' frames.
const PathFrontchannelLogout = "/oauth2/logout/frontchannel"

// FrontchannelPage is what that page shows, in the reader's language.
type FrontchannelPage struct {
	Language string
	Frames   []string
	Next     string
	Title    string
	Body     string
	Continue string
}

// frontchannelFrames are the addresses to load for the browser's session:
// each of its applications that asked to be told in the browser.
func (s *Service) frontchannelFrames(ctx context.Context, sessionToken string) ([]string, error) {
	session, err := s.SessionFor(ctx, sessionToken)
	if err != nil || session == nil {
		return nil, err
	}

	targets, err := s.store.FrontchannelTargets(ctx, session.Record.ID)
	if err != nil {
		return nil, err
	}

	frames := make([]string, 0, len(targets))
	for _, target := range targets {
		separator := "?"
		if strings.Contains(target.URI, "?") {
			separator = "&"
		}
		frames = append(frames, target.URI+separator+url.Values{
			"iss": {s.issuer}, "sid": {session.Record.ID.String()},
		}.Encode())
	}

	return frames, nil
}

// frontchannelPage keeps the frames and where to go after, and returns the
// page's address. The handle in it is good for one showing.
func (s *Service) frontchannelPage(ctx context.Context, frames []string, next string, client Client) (string, error) {
	handle, hash, err := model.NewSecret()
	if err != nil {
		return "", err
	}

	logout := model.FrontchannelLogout{
		HandleHash: hash,
		Frames:     frames,
		Next:       next,
		Language:   client.Language,
		ExpiresAt:  s.now().Add(model.FrontchannelLogoutLifetime),
	}
	if err := s.store.CreateFrontchannelLogout(ctx, &logout); err != nil {
		return "", err
	}

	return withQuery(s.issuer+PathFrontchannelLogout, url.Values{"logout": {handle}}), nil
}

// Frontchannel is the page for a logout, once. A handle that is unknown,
// spent or expired has nothing to show: the caller sends the browser to the
// signed-out page.
func (s *Service) Frontchannel(ctx context.Context, handle string) (*FrontchannelPage, error) {
	logout, err := s.store.ClaimFrontchannelLogout(ctx, model.HashSecret(handle), s.now())
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, store.ErrAlreadyUsed):
		return nil, ErrRequestExpired
	case err != nil:
		return nil, err
	}

	text := s.textIn(ctx, logout.Language)
	language := logout.Language
	if language == "" {
		language = "en"
	}

	return &FrontchannelPage{
		Language: language,
		Frames:   logout.Frames,
		Next:     logout.Next,
		Title:    text["frontchannel.head"],
		Body:     text["frontchannel.body"],
		Continue: text["frontchannel.continue"],
	}, nil
}

// SignedOutPage is where a browser goes when there is no logout page to show.
func (s *Service) SignedOutPage() string {
	return s.accountURL + PageLoggedOut
}
