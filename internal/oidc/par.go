package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"loginer/internal/model"
	"loginer/internal/store"
)

// Pushed authorization requests (RFC 9126). A client sends its authorization
// request server to server, authenticated as at the token endpoint, and gets
// a request_uri back; the browser then carries only that and the client_id,
// so nothing of the request can be read or changed on the way.

// PathPushedAuthorization is where a client pushes its request.
const PathPushedAuthorization = "/oauth2/par"

// PushedRequest is the answer to a push.
type PushedRequest struct {
	RequestURI string `json:"request_uri"`
	ExpiresIn  int    `json:"expires_in"`
}

// PushAuthorization checks an authorization request as the authorization
// endpoint would and keeps it for the browser. Every refusal is the client's
// to read, since no browser is involved yet.
func (s *Service) PushAuthorization(ctx context.Context, client ClientAuth, p AuthorizeParams) (*PushedRequest, error) {
	app, err := s.authenticate(ctx, client)
	if err != nil {
		return nil, err
	}

	switch {
	case p.RequestURI != "":
		return nil, oauthError(ErrInvalidRequest, "request_uri cannot itself be pushed")
	case p.ClientID != "" && p.ClientID != app.ClientID:
		return nil, oauthError(ErrInvalidRequest, "client_id does not match the client that authenticated")
	}
	p.ClientID = app.ClientID

	if problem := s.redirectProblem(app, p); problem != "" {
		return nil, oauthError(ErrInvalidRequest, problem)
	}
	code, description, err := s.requestProblem(ctx, app, p)
	if err != nil {
		return nil, err
	}
	if code != "" {
		return nil, oauthError(code, description)
	}

	params, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	handle, hash, err := model.NewSecret()
	if err != nil {
		return nil, err
	}

	pushed := model.PushedRequest{
		HandleHash:    hash,
		ApplicationID: app.ID,
		Params:        string(params),
		ExpiresAt:     s.now().Add(model.PushedRequestLifetime),
	}
	if err := s.store.CreatePushedRequest(ctx, &pushed); err != nil {
		return nil, err
	}

	return &PushedRequest{
		RequestURI: model.RequestURIPrefix + handle,
		ExpiresIn:  int(model.PushedRequestLifetime.Seconds()),
	}, nil
}

// takePushed swaps a request_uri for the request pushed under it, once. On a
// bad one it returns the error page to send the browser to instead.
func (s *Service) takePushed(ctx context.Context, p AuthorizeParams) (*AuthorizeParams, string, error) {
	refused := s.errorPage(ErrInvalidRequest, "request_uri is not valid or has expired; push the request again")

	handle, ok := strings.CutPrefix(p.RequestURI, model.RequestURIPrefix)
	if !ok || handle == "" {
		return nil, refused, nil
	}

	pushed, err := s.store.ClaimPushedRequest(ctx, model.HashSecret(handle), s.now())
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, store.ErrAlreadyUsed):
		return nil, refused, nil
	case err != nil:
		return nil, "", err
	}

	// The request_uri is for the client that pushed it, and no other.
	if pushed.Application == nil || pushed.Application.ClientID != p.ClientID {
		return nil, refused, nil
	}

	var taken AuthorizeParams
	if err := json.Unmarshal([]byte(pushed.Params), &taken); err != nil {
		return nil, "", err
	}

	return &taken, "", nil
}
