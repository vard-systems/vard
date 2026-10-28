// Package api builds the two HTTP servers and the table of which handler
// answers which path.
//
// The public server is the OAuth 2.0 / OpenID Connect provider and the account
// API, meant to face the internet. The admin server is the admin API the
// console calls, meant to be reachable only where administrators work. A path
// exists on only one of them.
//
// Handlers live one package per subject, each with handler.go, request.go,
// response.go and validation.go.
package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"vard/internal/api/account"
	"vard/internal/api/activity"
	"vard/internal/api/adminauth"
	"vard/internal/api/adminroles"
	"vard/internal/api/admins"
	"vard/internal/api/apis"
	"vard/internal/api/applications"
	"vard/internal/api/caching"
	"vard/internal/api/fields"
	"vard/internal/api/flows"
	"vard/internal/api/keys"
	"vard/internal/api/kit/audit"
	"vard/internal/api/kit/clientcert"
	"vard/internal/api/kit/cors"
	"vard/internal/api/kit/csrf"
	"vard/internal/api/kit/middleware"
	"vard/internal/api/kit/ratelimit"
	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/session"
	"vard/internal/api/languages"
	"vard/internal/api/mailsettings"
	"vard/internal/api/mfa"
	"vard/internal/api/oauth"
	"vard/internal/api/organization"
	"vard/internal/api/otp"
	"vard/internal/api/profile"
	"vard/internal/api/roles"
	"vard/internal/api/setup"
	"vard/internal/api/social"
	"vard/internal/api/sso"
	"vard/internal/api/steps"
	"vard/internal/api/twostep"
	"vard/internal/api/users"
	"vard/internal/api/usersessions"
	"vard/internal/auth"
	"vard/internal/brand"
	"vard/internal/cache"
	"vard/internal/config"
	"vard/internal/jose"
	"vard/internal/model"
	"vard/internal/oidc"
	"vard/internal/store"
)

// tokenLimitMultiple loosens the provider's endpoints relative to the sign-in
// pages: one application backend may refresh tokens for a whole company from
// one address.
const tokenLimitMultiple = 10

func NewPublic(cfg config.Config, log *slog.Logger, provider *oidc.Service, shared *cache.Redis) (*gin.Engine, error) {
	r, err := engine(cfg, log)
	if err != nil {
		return nil, err
	}

	certificate, err := clientcert.New(cfg.MTLSURL != "", cfg.TrustedProxies)
	if err != nil {
		return nil, err
	}

	registerPublicRoutes(r, publicHandlers{
		certificate: certificate,
		oauth:       oauth.New(provider, log, cfg.SecureUserCookies),
		account:     account.New(provider, log, cfg.SecureUserCookies),
		twoStep:     twostep.New(provider, log),
		profile:     profile.New(provider, log),
		steps:       steps.New(provider, log),
		limit:       ratelimit.New(cfg.RateLimit).Shared(shared.SessionDB(), "public").Middleware(),
		tokens:      ratelimit.New(cfg.RateLimit*tokenLimitMultiple).Shared(shared.SessionDB(), "token").Middleware(),
		csrf:        csrf.New(allowed(cfg.AccountURL, cfg.CORSOrigins)),
	})

	return r, nil
}

func NewAdmin(cfg config.Config, st *store.Store, sealer *jose.Sealer, log *slog.Logger, provider *oidc.Service, shared *cache.Redis) (*gin.Engine, error) {
	r, err := engine(cfg, log)
	if err != nil {
		return nil, err
	}

	service := auth.New(st, sealer, log, adminIssuer(cfg.AdminURL))
	recorder := audit.New(st, log)

	registerAdminRoutes(r, service, provider, log, adminHandlers{
		auth:         adminauth.New(service, st, recorder, log, cfg.SecureAdminCookies),
		mfa:          mfa.New(service, log),
		setup:        setup.New(st, log),
		users:        users.New(st, recorder, log),
		userSessions: usersessions.New(st, recorder, log),
		fields:       fields.New(st, recorder, log),
		roles:        roles.New(st, recorder, log),
		admins:       admins.New(st, service, recorder, log),
		adminRoles:   adminroles.New(st, recorder, log),
		applications: applications.New(st, recorder, log, cfg.Issuer, applications.MutualTLS{On: cfg.MTLSURL != "", Authorities: cfg.MTLSCAFile != ""}),
		organization: organization.New(st, recorder, log),
		social:       social.New(st, sealer, recorder, log, cfg.Issuer),
		sso:          sso.New(st, sealer, provider, recorder, log, cfg.Issuer),
		flows:        flows.New(st, recorder, log),
		languages:    languages.New(st, recorder, log),
		mail:         mailsettings.New(st, sealer, recorder, log),
		otp:          otp.New(st, recorder, log),
		apis:         apis.New(st, recorder, log, cfg.Issuer),
		activity:     activity.New(st, log),
		keys:         keys.New(provider, recorder, log),
		caching:      caching.New(shared, recorder, log),
		limit:        ratelimit.New(cfg.RateLimit).Shared(shared.SessionDB(), "admin").Middleware(),
		csrf:         csrf.New(allowed(cfg.AdminURL, cfg.CORSOrigins)),
	})

	return r, nil
}

func adminIssuer(adminURL string) string {
	if host := strings.TrimPrefix(strings.TrimPrefix(config.Origin(adminURL), "https://"), "http://"); host != "" {
		return brand.Name + " (" + host + ")"
	}
	return brand.Name
}

func engine(cfg config.Config, log *slog.Logger) (*gin.Engine, error) {
	r := gin.New()

	// Gin believes every X-Forwarded-For header unless told otherwise, which
	// would let any caller choose the address the log records for it.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("trusted proxies: %w", err)
	}

	r.Use(middleware.Chain(log, cors.New(cfg.CORSOrigins))...)

	return r, nil
}

func allowed(appURL string, extra []string) []string {
	return append([]string{config.Origin(appURL)}, extra...)
}

type publicHandlers struct {
	// certificate reads the TLS client certificate, on the endpoints the
	// mutual TLS host serves (oidc.MTLSPaths).
	certificate gin.HandlerFunc
	oauth       *oauth.Handler
	account     *account.Handler
	twoStep     *twostep.Handler
	profile     *profile.Handler
	steps       *steps.Handler
	limit       gin.HandlerFunc
	tokens      gin.HandlerFunc
	csrf        gin.HandlerFunc
}

type adminHandlers struct {
	auth         *adminauth.Handler
	mfa          *mfa.Handler
	setup        *setup.Handler
	users        *users.Handler
	userSessions *usersessions.Handler
	fields       *fields.Handler
	roles        *roles.Handler
	admins       *admins.Handler
	adminRoles   *adminroles.Handler
	applications *applications.Handler
	apis         *apis.Handler
	organization *organization.Handler
	social       *social.Handler
	sso          *sso.Handler
	flows        *flows.Handler
	languages    *languages.Handler
	mail         *mailsettings.Handler
	otp          *otp.Handler
	activity     *activity.Handler
	keys         *keys.Handler
	caching      *caching.Handler
	limit        gin.HandlerFunc
	csrf         gin.HandlerFunc
}

// registerPublicRoutes mounts every public route. What a path does is in the
// handler, but that a path exists — and on which server — is only ever here.
func registerPublicRoutes(r *gin.Engine, h publicHandlers) {
	r.GET("/healthz", health)

	r.GET(oidc.PathDiscovery, h.oauth.Discovery)
	r.GET(oidc.PathJWKS, h.oauth.JWKS)
	// Authorize writes a row per request, so it shares the token endpoints'
	// loose per-address limit.
	r.GET(oidc.PathAuthorize, h.tokens, h.oauth.Authorize)
	r.POST(oidc.PathAuthorize, h.tokens, h.oauth.Authorize)
	r.POST(oidc.PathToken, h.tokens, h.certificate, h.oauth.Token)
	r.POST(oidc.PathPushedAuthorization, h.tokens, h.certificate, h.oauth.PushAuthorization)
	r.POST(oidc.PathDeviceAuthorization, h.tokens, h.certificate, h.oauth.StartDevice)
	r.POST(oidc.PathRegistration, h.tokens, h.oauth.RegisterClient)
	r.GET(oidc.PathUserInfo, h.certificate, h.oauth.UserInfo)
	r.POST(oidc.PathUserInfo, h.certificate, h.oauth.UserInfo)
	r.GET(oidc.PathLogout, h.oauth.Logout)
	r.POST(oidc.PathLogout, h.oauth.Logout)
	r.GET(oidc.PathFrontchannelLogout, h.oauth.FrontchannelLogout)
	r.POST(oidc.PathRevoke, h.tokens, h.certificate, h.oauth.Revoke)
	r.POST(oidc.PathIntrospect, h.tokens, h.certificate, h.oauth.Introspect)

	// Social sign-in. The callback also answers POST and sits outside CSRF
	// because Apple posts its answer from its own form.
	r.GET(oidc.PathSocialStart, h.limit, h.oauth.SocialStart)
	r.GET(oidc.PathSocialCallback, h.oauth.SocialCallback)
	r.POST(oidc.PathSocialCallback, h.oauth.SocialCallback)

	// Enterprise SSO. The SAML ACS takes the provider's posted form, so it is
	// outside CSRF, and the browser then comes back to it with a GET; start
	// and ACS are rate limited because anyone can call them.
	r.GET(oidc.PathSSOStart, h.limit, h.oauth.SSOStart)
	r.GET(oidc.PathSSOCallback, h.oauth.SSOCallback)
	r.POST(oidc.PathSSOACS, h.limit, h.oauth.SSOAssertion)
	r.GET(oidc.PathSSOACS, h.limit, h.oauth.SSOAssertionReturn)
	r.GET(oidc.PathSSOMetadata, h.oauth.SSOMetadata)

	// The organisation's logo, for the sign-in pages: anyone may see it, and
	// the path is the provider's, which puts it in their addresses.
	r.GET(oidc.PathOrganizationLogo, h.account.OrganizationLogo)

	// Everything under /api/v1 is called by the id app with the user's
	// session cookie, so it only takes changes from the id app's origin.
	v1 := r.Group("/api/v1", h.csrf)
	{
		accounts := v1.Group("/account")
		accounts.GET("/organization", h.account.Organization)
		accounts.GET("/social-providers", h.account.SocialProviders)
		accounts.GET("/sso", h.account.SSOButtons)
		accounts.POST("/sso/discover", h.limit, h.account.DiscoverSSO)
		accounts.GET("/requests/:handle", h.account.Request)
		accounts.GET("/login-options", h.account.LoginOptions)
		accounts.GET("/languages", h.account.Languages)
		accounts.GET("/languages/:code", h.account.LanguageText)
		accounts.GET("/applications/:client_id", h.account.Application)
		accounts.POST("/login", h.limit, h.account.Login)
		accounts.POST("/login/code", h.limit, h.account.Code)
		accounts.POST("/login/code/resend", h.limit, h.account.ResendCode)
		accounts.GET("/login/second-factor", h.account.SecondFactor)
		accounts.POST("/login/second-factor", h.limit, h.account.SubmitSecondFactor)
		accounts.POST("/login/passkey", h.limit, h.account.BeginPasskey)
		accounts.POST("/login/passkey/finish", h.limit, h.account.FinishPasskey)
		accounts.POST("/register", h.limit, h.account.Register)
		accounts.POST("/forgot-password", h.limit, h.account.ForgotPassword)
		accounts.GET("/reset-password", h.account.CheckReset)
		accounts.POST("/reset-password", h.limit, h.account.ResetPassword)
		accounts.POST("/verify-email", h.limit, h.account.VerifyEmail)
		accounts.POST("/logout", h.account.Logout)

		own := accounts.Group("", h.account.RequireSession)
		own.GET("/me", h.profile.Me)
		own.PATCH("/me", h.profile.UpdateMe)
		own.POST("/password", h.limit, h.profile.ChangePassword)
		own.POST("/email", h.limit, h.profile.ChangeEmail)
		own.GET("/sessions", h.profile.Sessions)
		own.DELETE("/sessions/:id", h.profile.EndSession)
		own.GET("/connected-applications", h.profile.Applications)
		own.DELETE("/connected-applications/:client_id", h.profile.Disconnect)
		own.GET("/consent", h.steps.Consent)
		own.POST("/consent", h.steps.AnswerConsent)
		own.POST("/agreements", h.steps.AnswerAgreements)
		own.GET("/device", h.limit, h.steps.Device)
		own.POST("/device", h.limit, h.steps.AnswerDevice)
		own.GET("/two-step", h.twoStep.TwoStep)
		own.POST("/two-step/authenticator", h.limit, h.twoStep.BeginAuthenticator)
		own.POST("/two-step/authenticator/confirm", h.limit, h.twoStep.ConfirmAuthenticator)
		own.POST("/two-step/passkeys", h.limit, h.twoStep.BeginPasskey)
		own.POST("/two-step/passkeys/finish", h.twoStep.FinishPasskey)
		own.DELETE("/two-step/factors/:id", h.limit, h.twoStep.RemoveFactor)
		own.POST("/two-step/recovery-codes", h.limit, h.twoStep.NewRecoveryCodes)
	}

	r.NoRoute(notFound)
}

func registerAdminRoutes(r *gin.Engine, service *auth.Service, provider *oidc.Service, log *slog.Logger, h adminHandlers) {
	r.GET("/healthz", health)

	// Everything here is called by the console with the administrator's session
	// cookie, so it only takes changes from the console's origin.
	v1 := r.Group("/api/v1", h.csrf)
	{
		// Setup and sign-in cannot require a session. Setup is refused once any
		// administrator exists.
		v1.GET("/admin/setup", h.setup.Status)
		v1.POST("/admin/setup", h.limit, h.setup.Create)
		v1.POST("/admin/auth/login", h.limit, h.auth.Login)

		v1.GET("/admin/auth/session", h.auth.State)
		v1.POST("/admin/auth/mfa", h.limit, h.auth.VerifyMFA)
		v1.POST("/admin/auth/logout", h.auth.Logout)

		setup := v1.Group("/admin/mfa", session.RequireSetup(service))
		setup.GET("", h.mfa.Status)
		setup.POST("/totp", h.limit, h.mfa.Begin)
		setup.POST("/totp/confirm", h.limit, h.mfa.Confirm)

		signedIn := v1.Group("/admin", session.Require(service))
		signedIn.GET("/me", h.auth.Me)
		// Any administrator: the header shows the organisation's logo.
		signedIn.GET("/organization/logo", h.organization.Logo)
		signedIn.PATCH("/me", h.limit, h.auth.UpdateMe)
		signedIn.POST("/me/password", h.limit, h.auth.ChangePassword)

		signedIn.DELETE("/mfa/totp", h.limit, h.mfa.Disable)
		signedIn.POST("/mfa/recovery-codes", h.limit, h.mfa.RecoveryCodes)
		signedIn.GET("/sessions", h.auth.Sessions)
		signedIn.DELETE("/sessions/:id", h.auth.EndSession)
		signedIn.DELETE("/sessions", h.auth.EndOtherSessions)

		// Everything below is guarded by the administrator's permissions;
		// the panel hiding a control is not the check. These routes also
		// accept an admin API access token whose scopes stand in for roles;
		// routes about the signed-in person, and super-admin routes, stay
		// session-only.
		managed := v1.Group("/admin", session.RequireAny(service, provider, log))

		activity := managed.Group("", session.Can(model.PermActivityRead))
		activity.GET("/overview", h.activity.Overview)
		activity.GET("/logs", h.activity.Logs)
		activity.GET("/logs/export", h.activity.Export)

		readUsers := managed.Group("", session.Can(model.PermUsersRead))
		readUsers.GET("/users", h.users.List)
		readUsers.GET("/users/:id", h.users.Get)
		readUsers.GET("/users/:id/roles", h.users.Roles)
		readUsers.GET("/users/:id/role-mappings", h.users.RoleMappings)
		readUsers.GET("/user-fields", h.fields.List)
		readUsers.GET("/user-sessions", h.userSessions.List)

		writeUsers := managed.Group("", session.Can(model.PermUsersWrite))
		writeUsers.POST("/users", h.users.Create)
		writeUsers.PATCH("/users/:id", h.users.Update)
		writeUsers.DELETE("/users/:id", h.users.Delete)
		writeUsers.DELETE("/users/:id/social-accounts/:identity", h.users.Disconnect)
		writeUsers.DELETE("/users/:id/two-step", h.users.ResetTwoStep)
		writeUsers.DELETE("/users/:id/sessions", h.userSessions.SignOutUser)
		writeUsers.DELETE("/user-sessions/:id", h.userSessions.End)

		writeFields := managed.Group("", session.Can(model.PermUserFieldsWrite))
		writeFields.POST("/user-fields", h.fields.Create)
		writeFields.PATCH("/user-fields/:id", h.fields.Update)
		writeFields.DELETE("/user-fields/:id", h.fields.Delete)

		managed.GET("/organization", session.Can(model.PermOrganizationRead), h.organization.Get)
		managed.PATCH("/organization", session.Can(model.PermOrganizationWrite), h.organization.Update)
		managed.PUT("/organization/logo", session.Can(model.PermOrganizationWrite), h.organization.SetLogo)
		managed.DELETE("/organization/logo", session.Can(model.PermOrganizationWrite), h.organization.RemoveLogo)

		readSocial := managed.Group("", session.Can(model.PermSocialRead))
		readSocial.GET("/social-providers", h.social.List)
		readSocial.GET("/social-providers/:id", h.social.Get)

		// Reading a secret back takes the permission that could replace
		// it, and is recorded like a change.
		writeSocial := managed.Group("", session.Can(model.PermSocialWrite))
		writeSocial.GET("/social-providers/:id/secret", h.social.Secret)
		writeSocial.POST("/social-providers", h.social.Create)
		writeSocial.PATCH("/social-providers/:id", h.social.Update)
		writeSocial.DELETE("/social-providers/:id", h.social.Delete)

		readSSO := managed.Group("", session.Can(model.PermSSORead))
		readSSO.GET("/sso-connections", h.sso.List)
		readSSO.GET("/sso-connections/:id", h.sso.Get)
		readSSO.GET("/sso-connections/:id/domains", h.sso.Domains)

		writeSSO := managed.Group("", session.Can(model.PermSSOWrite))
		writeSSO.POST("/sso-connections", h.sso.Create)
		writeSSO.POST("/sso-connections/test", h.sso.Test)
		writeSSO.PATCH("/sso-connections/:id", h.sso.Update)
		writeSSO.DELETE("/sso-connections/:id", h.sso.Delete)
		writeSSO.POST("/sso-connections/:id/refresh-metadata", h.sso.RefreshMetadata)
		writeSSO.POST("/sso-connections/:id/domains/:domain/check", h.limit, h.sso.CheckDomain)

		readFlows := managed.Group("", session.Can(model.PermLoginFlowsRead))
		readFlows.GET("/login-flows", h.flows.List)
		readFlows.GET("/login-flows/:id", h.flows.Get)

		writeFlows := managed.Group("", session.Can(model.PermLoginFlowsWrite))
		writeFlows.POST("/login-flows", h.flows.Create)
		writeFlows.PATCH("/login-flows/:id", h.flows.Update)
		writeFlows.DELETE("/login-flows/:id", h.flows.Delete)

		readLanguages := managed.Group("", session.Can(model.PermLanguagesRead))
		readLanguages.GET("/languages", h.languages.List)
		readLanguages.GET("/languages/:code/translations/:app", h.languages.Translation)

		writeLanguages := managed.Group("", session.Can(model.PermLanguagesWrite))
		writeLanguages.POST("/languages", h.languages.Create)
		writeLanguages.PATCH("/languages/:code", h.languages.Update)
		writeLanguages.DELETE("/languages/:code", h.languages.Delete)
		writeLanguages.PUT("/languages/:code/translations/:app", h.languages.SaveTranslation)

		// Applications and their roles. Routes only check the administrator
		// reaches some application; each handler checks the one requested.
		apps := managed.Group("", session.CanAnywhere(model.PermApplicationsRead))
		apps.GET("/applications", h.applications.List)
		apps.GET("/applications/:id", h.applications.Get)
		apps.PATCH("/applications/:id", h.applications.Update)
		apps.POST("/applications/:id/secret", h.applications.RotateSecret)
		apps.GET("/applications/:id/apis", h.applications.APIAccess)
		apps.PUT("/applications/:id/apis/:api", h.applications.AuthorizeAPI)
		apps.DELETE("/applications/:id/apis/:api", h.applications.RevokeAPI)
		apps.POST("/applications/:id/token-preview", h.applications.TokenPreview)

		registerApps := managed.Group("", session.Can(model.PermApplicationsWrite))
		registerApps.POST("/applications", h.applications.Create)
		registerApps.DELETE("/applications/:id", h.applications.Delete)

		// User roles, global and per application. Handlers narrow to the
		// scopes the administrator reaches and check writes against the
		// role's scope.
		roles := managed.Group("", session.CanAnywhere(model.PermUsersRead, model.PermApplicationsRead))
		roles.GET("/user-roles", h.roles.List)
		roles.GET("/user-roles/:id", h.roles.Get)

		writeRoles := managed.Group("", session.CanAnywhere(model.PermUserRolesWrite))
		writeRoles.POST("/user-roles", h.roles.Create)
		writeRoles.PATCH("/user-roles/:id", h.roles.Update)
		writeRoles.DELETE("/user-roles/:id", h.roles.Delete)

		assign := managed.Group("", session.CanAnywhere(model.PermRoleAssignmentsWrite))
		assign.POST("/users/:id/role-mappings", h.users.AssignRoles)
		assign.DELETE("/users/:id/role-mappings/:role", h.users.UnassignRole)

		readAPIs := managed.Group("", session.CanAnywhere(model.PermAPIsRead, model.PermApplicationsWrite))
		readAPIs.GET("/apis", h.apis.List)
		readAPIs.GET("/apis/:id", h.apis.Get)
		readAPIs.GET("/apis/:id/applications", h.apis.Applications)

		// An API's log names every application, visible or not, so it needs
		// apis.read itself.
		managed.GET("/apis/:id/logs", session.Can(model.PermAPIsRead), h.apis.Logs)

		writeAPIs := managed.Group("", session.Can(model.PermAPIsWrite))
		writeAPIs.POST("/apis", h.apis.Create)
		writeAPIs.PATCH("/apis/:id", h.apis.Update)
		writeAPIs.DELETE("/apis/:id", h.apis.Delete)

		// The administrators themselves, and their roles: a super
		// admin's alone, whatever other roles grant.
		super := signedIn.Group("", session.RequireSuperAdmin())
		super.GET("/security", h.admins.Security)
		super.PATCH("/security", h.admins.UpdateSecurity)

		super.GET("/admins", h.admins.List)
		super.POST("/admins", h.admins.Create)
		super.GET("/admins/:id", h.admins.Get)
		super.PATCH("/admins/:id", h.admins.Update)
		super.DELETE("/admins/:id", h.admins.Delete)
		super.DELETE("/admins/:id/mfa", h.admins.ResetMFA)

		super.GET("/mail", h.mail.Get)
		super.PATCH("/mail", h.mail.Update)
		super.POST("/mail/test", h.limit, h.mail.Test)
		super.GET("/mail/content", h.mail.Content)
		super.PUT("/mail/content/:code", h.mail.SaveContent)

		super.GET("/otp", h.otp.Get)
		super.PATCH("/otp", h.otp.Update)

		super.GET("/signing-keys", h.keys.List)
		super.POST("/signing-keys/rotate", h.limit, h.keys.Rotate)

		super.GET("/cache", h.caching.Overview)
		super.GET("/cache/:database/keys", h.caching.Keys)
		super.GET("/cache/:database/key", h.caching.Key)
		super.PUT("/cache/:database/key", h.caching.UpdateKey)
		super.DELETE("/cache/:database/key", h.caching.DeleteKey)
		super.POST("/cache/:database/groups/:group/clear", h.caching.ClearGroup)
		super.DELETE("/cache/:database", h.caching.Flush)

		super.GET("/admin-permissions", h.adminRoles.Permissions)
		super.GET("/admin-roles", h.adminRoles.List)
		super.POST("/admin-roles", h.adminRoles.Create)
		super.PATCH("/admin-roles/:id", h.adminRoles.Update)
		super.DELETE("/admin-roles/:id", h.adminRoles.Delete)
	}

	r.NoRoute(notFound)
}

// health says the server is up.
func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func notFound(c *gin.Context) {
	respond.Fail(c, respond.NotFoundAny)
}
