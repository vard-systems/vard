---
title: "Public API"
description: "Every endpoint your application can call: the OAuth 2.0 and OpenID Connect provider, and the account API."
order: 0
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

| | |
| --- | --- |
| Base URL | Your issuer, such as `https://id.example.com` — the examples write it `$ISSUER` |
| Format | JSON, except the token, revocation and introspection endpoints, which take a form |
| Errors | A status and a stable `code` — see [Errors](/reference/errors) |
| OpenAPI | [Download](/openapi/public.json), or `GET $ISSUER/.well-known/openapi.json` |

## [Health](/reference/public/health)

Whether the server is up, for load balancers and uptime checks.

| Endpoint | |
| --- | --- |
| [`GET /healthz`](/reference/public/health#get-healthz) | Says the server is up. |

## [OAuth 2.0 and OpenID Connect](/reference/public/oauth)

The endpoints your application signs users in with, and gets and checks tokens from.

| Endpoint | |
| --- | --- |
| [`GET /.well-known/openid-configuration`](/reference/public/oauth#get-well-known-openid-configuration) | Answers the OpenID Connect discovery document. |
| [`GET /.well-known/jwks.json`](/reference/public/oauth#get-well-known-jwks-json) | Answers the public keys tokens are signed with. |
| [`GET /oauth2/authorize`](/reference/public/oauth#get-oauth2-authorize) | Starts a sign-in for an application, and redirects: to the sign-in page, or straight back with a code for a user already signed in. |
| [`POST /oauth2/authorize`](/reference/public/oauth#post-oauth2-authorize) | Starts a sign-in for an application, and redirects: to the sign-in page, or straight back with a code for a user already signed in. |
| [`POST /oauth2/token`](/reference/public/oauth#post-oauth2-token) | Issues tokens. |
| [`GET /oauth2/userinfo`](/reference/public/oauth#get-oauth2-userinfo) | Answers the claims about the user an access token is for. |
| [`POST /oauth2/userinfo`](/reference/public/oauth#post-oauth2-userinfo) | Answers the claims about the user an access token is for. |
| [`GET /oauth2/logout`](/reference/public/oauth#get-oauth2-logout) | Signs the user out of this server, and redirects. |
| [`POST /oauth2/logout`](/reference/public/oauth#post-oauth2-logout) | Signs the user out of this server, and redirects. |
| [`POST /oauth2/revoke`](/reference/public/oauth#post-oauth2-revoke) | Revokes a refresh token. |
| [`POST /oauth2/introspect`](/reference/public/oauth#post-oauth2-introspect) | Says whether a token is active. |
| [`GET /oauth2/social/:slug/start`](/reference/public/oauth#get-oauth2-social-slug-start) | Sends the browser to a provider and leaves the state in a cookie, binding the callback to this browser. |
| [`GET /oauth2/social/:slug/callback`](/reference/public/oauth#get-oauth2-social-slug-callback) | Is where the provider sends the browser back. |
| [`POST /oauth2/social/:slug/callback`](/reference/public/oauth#post-oauth2-social-slug-callback) | Is where the provider sends the browser back. |
| [`GET /oauth2/sso/:slug/start`](/reference/public/oauth#get-oauth2-sso-slug-start) | Sends the browser to a connection's identity provider, passing the typed address as `login_hint`. |
| [`GET /oauth2/sso/:slug/callback`](/reference/public/oauth#get-oauth2-sso-slug-callback) | Is where an OpenID Connect provider sends the browser back to. |
| [`POST /oauth2/sso/:slug/acs`](/reference/public/oauth#post-oauth2-sso-slug-acs) | Is the assertion consumer service a SAML provider posts its response to. |
| [`GET /oauth2/sso/:slug/metadata`](/reference/public/oauth#get-oauth2-sso-slug-metadata) | Is this server's SAML metadata as one connection's service provider: the file an identity provider is set up from. |

## [OpenAPI document](/reference/public/reference)

The server's own OpenAPI document, for your tooling.

| Endpoint | |
| --- | --- |
| [`GET /.well-known/openapi.json`](/reference/public/reference#get-well-known-openapi-json) | Answers the OpenAPI document. |

## [Account](/reference/public/account)

The API behind the hosted sign-in pages and a user's account page. Applications do not usually call it.

| Endpoint | |
| --- | --- |
| [`GET /api/v1/account/organization`](/reference/public/account#get-api-v1-account-organization) | Describes the organisation the sign-in pages speak for. |
| [`GET /api/v1/account/social-providers`](/reference/public/account#get-api-v1-account-social-providers) | Lists the sign-in buttons, with only what a button needs. |
| [`GET /api/v1/account/sso`](/reference/public/account#get-api-v1-account-sso) | Lists the organisations' identity providers the sign-in page offers a button for. |
| [`POST /api/v1/account/sso/discover`](/reference/public/account#post-api-v1-account-sso-discover) | Says which connection an address signs in through, for "Sign in with SSO": the one that owns its domain, or no_sso_connection. |
| [`GET /api/v1/account/requests/:handle`](/reference/public/account#get-api-v1-account-requests-handle) | Describes a sign-in under way: the application it is for, as its sign-in page shows it. |
| [`GET /api/v1/account/login-options`](/reference/public/account#get-api-v1-account-login-options) | Returns the options of the login flow for the sign-in handle `request`, or the default flow. |
| [`GET /api/v1/account/languages`](/reference/public/account#get-api-v1-account-languages) | Says which languages these pages may be shown in, and which one somebody gets before they have chosen. |
| [`GET /api/v1/account/languages/:code`](/reference/public/account#get-api-v1-account-languages-code) | Returns the sign-in pages' text in one offered language with every key filled in. |
| [`GET /api/v1/account/applications/:client_id`](/reference/public/account#get-api-v1-account-applications-client-id) | Describes an application by its client id, for the signed-out page to offer a way back to it. |
| [`POST /api/v1/account/login`](/reference/public/account#post-api-v1-account-login) | Signs a user in and, for a sign-in under way, says where to go next. |
| [`POST /api/v1/account/login/code`](/reference/public/account#post-api-v1-account-login-code) | Finishes a sign-in that was waiting for an emailed code, and answers as the sign-in itself does: a cookie, and where to go next. |
| [`POST /api/v1/account/login/code/resend`](/reference/public/account#post-api-v1-account-login-code-resend) | Sends another code for a sign-in that is still waiting, and answers with the wait before the next one. |
| [`POST /api/v1/account/register`](/reference/public/account#post-api-v1-account-register) | Creates an account for a sign-in under way, and signs it in. |
| [`POST /api/v1/account/forgot-password`](/reference/public/account#post-api-v1-account-forgot-password) | Sends a reset link. |
| [`GET /api/v1/account/reset-password`](/reference/public/account#get-api-v1-account-reset-password) | Says whether a reset link still works. |
| [`POST /api/v1/account/reset-password`](/reference/public/account#post-api-v1-account-reset-password) | Sets a new password through a reset link. |
| [`POST /api/v1/account/verify-email`](/reference/public/account#post-api-v1-account-verify-email) | Uses a verification link. |
| [`POST /api/v1/account/logout`](/reference/public/account#post-api-v1-account-logout) | Signs the user out of this server in this browser. |
| [`GET /api/v1/account/me`](/reference/public/account#get-api-v1-account-me) | Returns the signed-in user. |
| [`PATCH /api/v1/account/me`](/reference/public/account#patch-api-v1-account-me) | Changes the signed-in user's name. |
| [`POST /api/v1/account/password`](/reference/public/account#post-api-v1-account-password) | Replaces the signed-in user's password. |
| [`POST /api/v1/account/email`](/reference/public/account#post-api-v1-account-email) | Starts moving the signed-in user to another address, given their current password. |
| [`GET /api/v1/account/sessions`](/reference/public/account#get-api-v1-account-sessions) | Lists where the signed-in user is signed in. |
| [`DELETE /api/v1/account/sessions/:id`](/reference/public/account#delete-api-v1-account-sessions-id) | Signs one of the user's other browsers out. |
| [`GET /api/v1/account/connected-applications`](/reference/public/account#get-api-v1-account-connected-applications) | Lists the applications that can still act for the user. |
| [`DELETE /api/v1/account/connected-applications/:client_id`](/reference/public/account#delete-api-v1-account-connected-applications-client-id) | Revokes an application's refresh tokens for the user. |
