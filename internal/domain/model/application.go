package model

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Application is an OAuth client. It follows RFC 7591 metadata and RFC 9700:
// exact redirect URI matching, no implicit or password grants, and PKCE for
// clients that cannot keep a secret. Its roles mean something only within it.
type Application struct {
	Base

	Name        string          `gorm:"size:100;not null" json:"name"`
	Description string          `gorm:"size:255;not null;default:''" json:"description"`
	Type        ApplicationType `gorm:"type:varchar(16);not null;index;check:chk_applications_type,type IN ('web','spa','native','m2m')" json:"type"`

	// LogoURL and ClientURI are shown on the sign-in pages, and on the
	// consent page when the application's login flow has that step.
	LogoURL    string `gorm:"size:512;not null;default:''" json:"logo_url"`
	WebsiteURL string `gorm:"size:512;not null;default:''" json:"website_url"`

	PrivacyURL string `gorm:"size:512;not null;default:''" json:"privacy_url"`
	TermsURL   string `gorm:"size:512;not null;default:''" json:"terms_url"`

	AllowRegistration bool `gorm:"not null" json:"allow_registration"`

	ClientID string `gorm:"size:64;uniqueIndex;not null" json:"client_id"`

	ClientSecretHash string     `gorm:"size:64;not null;default:''" json:"-"`
	SecretHint       string     `gorm:"size:8;not null;default:''" json:"secret_hint"`
	SecretCreatedAt  *time.Time `json:"secret_created_at"`

	TokenAuthMethod AuthMethod `gorm:"type:varchar(32);not null;check:chk_applications_token_auth_method,token_auth_method IN ('client_secret_basic','client_secret_post','none')" json:"token_auth_method"`

	GrantTypes             []string `gorm:"type:jsonb;serializer:json" json:"grant_types"`
	RedirectURIs           []string `gorm:"type:jsonb;serializer:json" json:"redirect_uris"`
	PostLogoutRedirectURIs []string `gorm:"type:jsonb;serializer:json" json:"post_logout_redirect_uris"`
	// AllowDevelopmentRedirects lets a native app register Expo Go's exp://
	// addresses while it is developed. They name a developer's machine on
	// their network, so a released app turns it off.
	AllowDevelopmentRedirects bool     `gorm:"not null;default:false" json:"allow_development_redirects"`
	Scopes                    []string `gorm:"type:jsonb;serializer:json" json:"scopes"`

	// BackchannelLogoutURI is told, server to server, when a session the
	// application signed someone in through ends (OpenID Connect
	// Back-Channel Logout 1.0). Empty, it is not told.
	BackchannelLogoutURI string `gorm:"size:512;not null;default:''" json:"backchannel_logout_uri"`
	// FrontchannelLogoutURI is loaded in a hidden frame of the browser that
	// signs out, with iss and sid (OpenID Connect Front-Channel Logout 1.0):
	// for an application that ends its session in the browser.
	FrontchannelLogoutURI string `gorm:"size:512;not null;default:''" json:"frontchannel_logout_uri"`

	RequirePKCE bool `gorm:"not null" json:"require_pkce"`
	// RequirePushedRequests refuses an authorization request the application
	// did not push first (RFC 9126 §6), so its parameters never travel in a
	// browser's address.
	RequirePushedRequests bool `gorm:"not null" json:"require_pushed_authorization_requests"`

	AccessTokenLifetime  int `gorm:"not null" json:"access_token_lifetime"`
	IDTokenLifetime      int `gorm:"not null" json:"id_token_lifetime"`
	RefreshTokenLifetime int `gorm:"not null" json:"refresh_token_lifetime"`

	AssertRoles           bool `gorm:"not null" json:"assert_roles"`
	RequireRoleAssignment bool `gorm:"not null;default:false" json:"require_role_assignment"`

	// A flow deleted leaves its applications on the default one.
	LoginFlowID *uuid.UUID `gorm:"type:uuid;index" json:"login_flow_id"`
	LoginFlow   *LoginFlow `gorm:"constraint:OnDelete:SET NULL" json:"-"`

	IsEnabled bool `gorm:"not null;index" json:"is_enabled"`
}

func (Application) TableName() string {
	return "applications"
}

type ApplicationType string

const (
	AppWeb    ApplicationType = "web"
	AppSPA    ApplicationType = "spa"
	AppNative ApplicationType = "native"
	AppM2M    ApplicationType = "m2m"
)

var ApplicationTypes = []ApplicationType{AppWeb, AppSPA, AppNative, AppM2M}

func (t ApplicationType) Valid() bool {
	return slices.Contains(ApplicationTypes, t)
}

func (t ApplicationType) Public() bool {
	return t == AppSPA || t == AppNative
}

type AuthMethod string

const (
	AuthClientSecretBasic AuthMethod = "client_secret_basic"
	AuthClientSecretPost  AuthMethod = "client_secret_post"
	AuthNone              AuthMethod = "none"
)

var AuthMethods = []AuthMethod{AuthClientSecretBasic, AuthClientSecretPost, AuthNone}

// The grants an application may use. The implicit and resource owner password
// grants are left out on purpose: RFC 9700 says not to use them.
const (
	GrantAuthorizationCode = "authorization_code"
	GrantRefreshToken      = "refresh_token"
	GrantClientCredentials = "client_credentials"
	// GrantDeviceCode is the device authorization grant (RFC 8628).
	GrantDeviceCode = "urn:ietf:params:oauth:grant-type:device_code"
)

var GrantTypes = []string{GrantAuthorizationCode, GrantRefreshToken, GrantClientCredentials, GrantDeviceCode}

const (
	ScopeOpenID        = "openid"
	ScopeProfile       = "profile"
	ScopeEmail         = "email"
	ScopeOfflineAccess = "offline_access"
	ScopeRoles         = "roles"
)

var Scopes = []string{ScopeOpenID, ScopeProfile, ScopeEmail, ScopeOfflineAccess, ScopeRoles}

const (
	// DefaultAccessTokenLifetime is short: an API checks an access token on
	// its own, so a revoked one is only refused by this server's endpoints
	// until it expires.
	DefaultAccessTokenLifetime  = 10 * 60
	DefaultIDTokenLifetime      = 60 * 60
	DefaultRefreshTokenLifetime = 30 * 24 * 60 * 60
)

// MaxAccessTokenLifetime is the longest an access token lives, an
// application's or an API's: how long a revocation has to be remembered.
const MaxAccessTokenLifetime = 24 * 60 * 60

const (
	minTokenLifetime        = 60
	maxAccessTokenLifetime  = MaxAccessTokenLifetime
	maxRefreshTokenLifetime = 365 * 24 * 60 * 60
)

func (a Application) ResponseTypes() []string {
	if slices.Contains(a.GrantTypes, GrantAuthorizationCode) {
		return []string{"code"}
	}

	return []string{}
}

// SignsUsersIn says the application has a grant that gets tokens for a user:
// through a browser, or through a device the user allows elsewhere.
func (a Application) SignsUsersIn() bool {
	return slices.Contains(a.GrantTypes, GrantAuthorizationCode) || slices.Contains(a.GrantTypes, GrantDeviceCode)
}

// HasSecret says the application authenticates with a client secret.
func (a Application) HasSecret() bool {
	return a.TokenAuthMethod == AuthClientSecretBasic || a.TokenAuthMethod == AuthClientSecretPost
}

// MustUsePKCE is whether an authorization request has to carry a
// code_challenge. A client with no credentials of its own always does
// (RFC 9700 §2.1.1, OAuth 2.1): nothing else ties the code to the app that
// asked for it, whatever RequirePKCE says.
func (a Application) MustUsePKCE() bool {
	return a.RequirePKCE || a.TokenAuthMethod == AuthNone
}

// AllowsRedirect is whether uri is one of the application's redirect URIs.
// Matching is exact, but a native app's loopback URI matches on any port: the
// app opens whichever port is free when it signs in (RFC 8252 §7.3).
// localhost is taken as well as 127.0.0.1 and [::1], though RFC 8252 §8.3
// advises against it, since a resolver may point it elsewhere; the docs and
// the SDKs use the addresses.
func (a Application) AllowsRedirect(uri string) bool {
	if slices.Contains(a.RedirectURIs, uri) {
		return true
	}
	if a.Type != AppNative {
		return false
	}

	sent, err := url.Parse(uri)
	if err != nil || sent.Scheme != "http" || !loopback(sent.Hostname()) {
		return false
	}

	for _, registered := range a.RedirectURIs {
		want, err := url.Parse(registered)
		if err != nil || want.Scheme != "http" || !loopback(want.Hostname()) {
			continue
		}
		if want.Hostname() == sent.Hostname() && want.Path == sent.Path && want.RawQuery == sent.RawQuery && sent.User == nil && sent.Fragment == "" {
			return true
		}
	}

	return false
}

func (a *Application) Normalize() {
	switch {
	case a.Type.Public():
		a.TokenAuthMethod = AuthNone
		a.RequirePKCE = true
	case a.TokenAuthMethod == "" || a.TokenAuthMethod == AuthNone:
		a.TokenAuthMethod = AuthClientSecretBasic
	}

	if a.Type != AppNative {
		a.AllowDevelopmentRedirects = false
	}

	if a.Type == AppM2M {
		a.AllowRegistration = false
		a.GrantTypes = []string{GrantClientCredentials}
		a.RedirectURIs = []string{}
		a.PostLogoutRedirectURIs = []string{}
		a.BackchannelLogoutURI = ""
		a.FrontchannelLogoutURI = ""
		a.Scopes = []string{}
		a.RequirePKCE = false
		a.RequireRoleAssignment = false
		a.RequirePushedRequests = false
	}

	a.GrantTypes = ordered(a.GrantTypes, GrantTypes)
	a.Scopes = ordered(a.Scopes, Scopes)
	a.RedirectURIs = tidyURIs(a.RedirectURIs)
	a.PostLogoutRedirectURIs = tidyURIs(a.PostLogoutRedirectURIs)
	a.BackchannelLogoutURI = strings.TrimSpace(a.BackchannelLogoutURI)
	a.FrontchannelLogoutURI = strings.TrimSpace(a.FrontchannelLogoutURI)
}

func (a Application) Validate() error {
	if !a.Type.Valid() {
		return fmt.Errorf("type must be one of: web, spa, native, m2m")
	}

	if !slices.Contains(AuthMethods, a.TokenAuthMethod) {
		return fmt.Errorf("token_auth_method must be one of: client_secret_basic, client_secret_post, none")
	}

	for _, grant := range a.GrantTypes {
		if !slices.Contains(GrantTypes, grant) {
			return fmt.Errorf("grant_types: %q is not offered (implicit and password grants are not supported)", grant)
		}
	}

	for _, scope := range a.Scopes {
		if !slices.Contains(Scopes, scope) {
			return fmt.Errorf("scopes: %q is not a scope", scope)
		}
	}

	code := slices.Contains(a.GrantTypes, GrantAuthorizationCode)
	device := slices.Contains(a.GrantTypes, GrantDeviceCode)

	switch a.Type {
	case AppM2M:
	case AppWeb, AppSPA, AppNative:
		if !code && !device {
			return fmt.Errorf("grant_types: a %s application signs users in, so it needs authorization_code or the device code grant", a.Type)
		}
		if a.Type.Public() && slices.Contains(a.GrantTypes, GrantClientCredentials) {
			return fmt.Errorf("grant_types: a public client has no secret, so it cannot use client_credentials")
		}
	}

	if slices.Contains(a.GrantTypes, GrantRefreshToken) && !code && !device {
		return fmt.Errorf("grant_types: refresh_token needs authorization_code or the device code grant")
	}

	if code && len(a.RedirectURIs) == 0 {
		return fmt.Errorf("redirect_uris: authorization_code needs at least one redirect URI")
	}

	for _, uri := range a.RedirectURIs {
		if err := a.checkRedirectURI(uri); err != nil {
			return fmt.Errorf("redirect_uris: %w", err)
		}
	}

	for _, uri := range a.PostLogoutRedirectURIs {
		if err := a.checkRedirectURI(uri); err != nil {
			return fmt.Errorf("post_logout_redirect_uris: %w", err)
		}
	}

	if a.BackchannelLogoutURI != "" {
		if err := checkLogoutURI(a.BackchannelLogoutURI); err != nil {
			return fmt.Errorf("backchannel_logout_uri: %w", err)
		}
	}

	if a.FrontchannelLogoutURI != "" {
		if err := checkLogoutURI(a.FrontchannelLogoutURI); err != nil {
			return fmt.Errorf("frontchannel_logout_uri: %w", err)
		}
	}

	// Plain http is fine for links, but logo_url is loaded as an image on an
	// https page, where browsers block http.
	links := []struct{ name, value string }{
		{"website_url", a.WebsiteURL},
		{"privacy_url", a.PrivacyURL},
		{"terms_url", a.TermsURL},
	}
	for _, link := range links {
		if link.value == "" {
			continue
		}
		if parsed, err := url.Parse(link.value); err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			return fmt.Errorf("%s must be an http or https URL", link.name)
		}
	}

	if a.LogoURL != "" {
		if parsed, err := url.Parse(a.LogoURL); err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("logo_url must be an https URL")
		}
	}

	lifetimes := []struct {
		name  string
		value int
		max   int
	}{
		{"access_token_lifetime", a.AccessTokenLifetime, maxAccessTokenLifetime},
		{"id_token_lifetime", a.IDTokenLifetime, maxAccessTokenLifetime},
		{"refresh_token_lifetime", a.RefreshTokenLifetime, maxRefreshTokenLifetime},
	}
	for _, lifetime := range lifetimes {
		if lifetime.value < minTokenLifetime || lifetime.value > lifetime.max {
			return fmt.Errorf("%s must be between %d and %d seconds", lifetime.name, minTokenLifetime, lifetime.max)
		}
	}

	return nil
}

// checkRedirectURI enforces RFC 9700: absolute, no fragment or wildcard, https
// except on loopback, and for native apps a private-use scheme (RFC 8252), or
// Expo Go's while they are developed.
func (a Application) checkRedirectURI(raw string) error {
	kind := a.Type

	if strings.Contains(raw, "*") {
		return fmt.Errorf("%q: wildcards are not allowed, redirect URIs are matched exactly", raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() {
		return fmt.Errorf("%q is not an absolute URI", raw)
	}

	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		return fmt.Errorf("%q must not have a fragment", raw)
	}

	switch parsed.Scheme {
	case "https":
		if parsed.Host == "" {
			return fmt.Errorf("%q has no host", raw)
		}
		return nil
	case "http":
		if loopback(parsed.Hostname()) {
			return nil
		}
		return fmt.Errorf("%q must use https, unless it is on localhost", raw)
	case "exp", "exps":
		// Expo Go, while a native app is developed: the scheme is Expo Go's
		// own, and PKCE, which a native app always uses, keeps a code that
		// reaches another app useless to it. The address is a developer's
		// machine, so only an app marked as in development may have one.
		switch {
		case kind != AppNative:
			return fmt.Errorf("%q must use https", raw)
		case !a.AllowDevelopmentRedirects:
			return fmt.Errorf("%q is Expo Go's, for development: allow development redirects on this application to register it, and turn them off before it is released", raw)
		}
		return nil
	default:
		// A private-use scheme is a reverse domain name, so it has a dot in
		// it and cannot be mistaken for javascript: or data:.
		if kind == AppNative && strings.Contains(parsed.Scheme, ".") {
			return nil
		}
		return fmt.Errorf("%q must use https", raw)
	}
}

// checkLogoutURI holds an address an application is told of a logout at:
// absolute, https except on loopback, and without a fragment (Back-Channel
// Logout 1.0 §2.2, Front-Channel Logout 1.0 §2). A query is allowed.
func checkLogoutURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return fmt.Errorf("%q is not an absolute URL", raw)
	}

	if parsed.Fragment != "" || strings.Contains(raw, "#") {
		return fmt.Errorf("%q must not have a fragment", raw)
	}

	switch {
	case parsed.Scheme == "https":
		return nil
	case parsed.Scheme == "http" && loopback(parsed.Hostname()):
		return nil
	default:
		return fmt.Errorf("%q must use https, unless it is on localhost", raw)
	}
}

func loopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func NewClientID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate client id: %w", err)
	}

	return hex.EncodeToString(b[:]), nil
}

var ErrNoSecret = errors.New("a public client has no secret")

func (a *Application) IssueSecret(now time.Time) (string, error) {
	if !a.HasSecret() {
		return "", ErrNoSecret
	}

	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate client secret: %w", err)
	}

	secret := base64.RawURLEncoding.EncodeToString(b[:])
	sum := sha256.Sum256([]byte(secret))

	a.ClientSecretHash = hex.EncodeToString(sum[:])
	a.SecretHint = secret[len(secret)-4:]
	a.SecretCreatedAt = &now

	return secret, nil
}

// CheckSecret reports whether a presented secret is the application's, in
// constant time.
func (a Application) CheckSecret(secret string) bool {
	if a.ClientSecretHash == "" {
		return false
	}

	sum := sha256.Sum256([]byte(secret))

	return subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(a.ClientSecretHash)) == 1
}

func ordered(values, catalog []string) []string {
	out := []string{}
	for _, known := range catalog {
		if slices.Contains(values, known) {
			out = append(out, known)
		}
	}

	for _, value := range values {
		if !slices.Contains(out, value) {
			out = append(out, value)
		}
	}

	return out
}

func tidyURIs(uris []string) []string {
	out := []string{}
	for _, uri := range uris {
		uri = strings.TrimSpace(uri)
		if uri != "" && !slices.Contains(out, uri) {
			out = append(out, uri)
		}
	}

	return out
}
