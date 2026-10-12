package oauth

import (
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"loginer/internal/model"
	"loginer/internal/oidc"
)

func authorizeParams(c *gin.Context) (oidc.AuthorizeParams, error) {
	if err := c.Request.ParseForm(); err != nil {
		return oidc.AuthorizeParams{}, err
	}

	form := c.Request.Form
	return oidc.AuthorizeParams{
		ClientID:            form.Get("client_id"),
		RedirectURI:         form.Get("redirect_uri"),
		ResponseType:        form.Get("response_type"),
		ResponseMode:        form.Get("response_mode"),
		Scope:               form.Get("scope"),
		State:               form.Get("state"),
		Nonce:               form.Get("nonce"),
		CodeChallenge:       form.Get("code_challenge"),
		CodeChallengeMethod: form.Get("code_challenge_method"),
		Audience:            form.Get("audience"),
		Prompt:              form.Get("prompt"),
		MaxAge:              form.Get("max_age"),
		LoginHint:           form.Get("login_hint"),
	}, nil
}

func logoutParams(c *gin.Context) (oidc.LogoutParams, error) {
	if err := c.Request.ParseForm(); err != nil {
		return oidc.LogoutParams{}, err
	}

	form := c.Request.Form
	return oidc.LogoutParams{
		IDTokenHint:           form.Get("id_token_hint"),
		ClientID:              form.Get("client_id"),
		PostLogoutRedirectURI: form.Get("post_logout_redirect_uri"),
		State:                 form.Get("state"),
	}, nil
}

func tokenParams(c *gin.Context) (oidc.TokenParams, error) {
	form, err := readForm(c)
	if err != nil {
		return oidc.TokenParams{}, err
	}

	auth, err := clientAuth(c, form)
	if err != nil {
		return oidc.TokenParams{}, err
	}

	return oidc.TokenParams{
		Client:       auth,
		GrantType:    form.Get("grant_type"),
		Code:         form.Get("code"),
		RedirectURI:  form.Get("redirect_uri"),
		CodeVerifier: form.Get("code_verifier"),
		RefreshToken: form.Get("refresh_token"),
		Scope:        form.Get("scope"),
		Audience:     form.Get("audience"),
	}, nil
}

func tokenOnlyParams(c *gin.Context) (oidc.ClientAuth, string, error) {
	form, err := readForm(c)
	if err != nil {
		return oidc.ClientAuth{}, "", err
	}

	auth, err := clientAuth(c, form)
	if err != nil {
		return oidc.ClientAuth{}, "", err
	}

	return auth, form.Get("token"), nil
}

func readForm(c *gin.Context) (url.Values, error) {
	if c.Request.Method != http.MethodPost {
		return nil, invalidRequest("this endpoint only accepts POST")
	}

	media, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if media != "application/x-www-form-urlencoded" {
		return nil, invalidRequest("the body must be application/x-www-form-urlencoded")
	}

	if err := c.Request.ParseForm(); err != nil {
		return nil, invalidRequest("the body could not be read")
	}

	// Parameters that mean something must not be repeated (RFC 6749 section 3.2).
	for key, values := range c.Request.PostForm {
		if len(values) > 1 {
			return nil, invalidRequest(key + " is repeated")
		}
	}

	return c.Request.PostForm, nil
}

// clientAuth reads how the client identified itself. Basic credentials are
// form-decoded (RFC 6749 2.3.1); using both methods at once is refused.
func clientAuth(c *gin.Context, form url.Values) (oidc.ClientAuth, error) {
	if id, secret, ok := c.Request.BasicAuth(); ok {
		if form.Get("client_secret") != "" {
			return oidc.ClientAuth{}, invalidRequest("send the client secret one way, not in both the header and the body")
		}

		decodedID, err1 := url.QueryUnescape(id)
		decodedSecret, err2 := url.QueryUnescape(secret)
		if err1 != nil || err2 != nil {
			return oidc.ClientAuth{}, &oidc.Error{Code: oidc.ErrInvalidClient, Description: "the Authorization header could not be read", Status: http.StatusUnauthorized}
		}

		if bodyID := form.Get("client_id"); bodyID != "" && bodyID != decodedID {
			return oidc.ClientAuth{}, invalidRequest("client_id in the body does not match the Authorization header")
		}

		return oidc.ClientAuth{ID: decodedID, Secret: decodedSecret, Method: model.AuthClientSecretBasic}, nil
	}

	if secret := form.Get("client_secret"); secret != "" {
		return oidc.ClientAuth{ID: form.Get("client_id"), Secret: secret, Method: model.AuthClientSecretPost}, nil
	}

	return oidc.ClientAuth{ID: form.Get("client_id"), Method: model.AuthNone}, nil
}

// bearer reads the access token from the Authorization header, or from the
// form of a POST (RFC 6750 section 2).
func bearer(c *gin.Context) (string, error) {
	if header := c.GetHeader("Authorization"); header != "" {
		scheme, token, found := strings.Cut(header, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			return "", &oidc.Error{Code: oidc.ErrInvalidToken, Description: "the Authorization header must be Bearer <access token>", Status: http.StatusUnauthorized}
		}
		return strings.TrimSpace(token), nil
	}

	if c.Request.Method == http.MethodPost {
		if err := c.Request.ParseForm(); err == nil {
			if token := c.Request.PostForm.Get("access_token"); token != "" {
				return token, nil
			}
		}
	}

	return "", &oidc.Error{Code: oidc.ErrInvalidToken, Description: "an access token is required", Status: http.StatusUnauthorized}
}
