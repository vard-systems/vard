---
title: "Errors"
description: "Every error code either server answers with, and what it means."
order: 10
icon: error-warning
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

Both servers answer an error the same way: a status, and a body with a stable `code` to act on, the English `error` to show a developer, and the `params` that fill in its `{placeholders}`.

```json
{ "error": "Too many attempts. Try again in 30 seconds.", "code": "rate_limited", "params": { "seconds": 30 } }
```

Match on `code`, never on `error`: the sentence may be reworded, and the sign-in pages show it in the reader's language. Any endpoint can also answer `500` with `internal` when the server itself fails; the cause is in its log, never in the answer.

The provider's own endpoints — token, userinfo, revocation, introspection — are the exception: they follow their RFCs and answer [OAuth errors](#oauth-errors).

## Public API {#public-api}

| Status | Code | Means |
| --- | --- | --- |
| 403 | <span id="account_scope_missing"></span>`account_scope_missing` | This access token needs the {scope} scope. |
| 401 | <span id="account_token_refused"></span>`account_token_refused` | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 404 | <span id="application_not_found"></span>`application_not_found` | There is no such application. |
| 413 | <span id="body_too_large"></span>`body_too_large` | The request is larger than this server accepts. |
| 403 | <span id="cross_origin"></span>`cross_origin` | Requests from {origin} are not allowed. |
| 403 | <span id="cross_site"></span>`cross_site` | Requests from other sites are not allowed. |
| 403 | <span id="forbidden"></span>`forbidden` | You do not have permission to do that. |
| 500 | <span id="internal"></span>`internal` | Something went wrong on our side. Try again in a moment. |
| 400 | <span id="invalid_body"></span>`invalid_body` | The request could not be read. Reload the page and try again. |
| 404 | <span id="language_not_found"></span>`language_not_found` | There is no such language. |
| 404 | <span id="no_sso_connection"></span>`no_sso_connection` | There is no single sign-on for that address. Sign in with your password instead. |
| 404 | <span id="not_found"></span>`not_found` | There is nothing here. |
| 401 | <span id="not_signed_in"></span>`not_signed_in` | You are not signed in. |
| 400 | <span id="password_too_short"></span>`password_too_short` | The password must be at least {min} characters. |
| 429 | <span id="rate_limited"></span>`rate_limited` | Too many attempts. Try again in {seconds} seconds. |
| 415 | <span id="unsupported_body"></span>`unsupported_body` | The request was sent in a form this server does not accept. |
| 400 | <span id="validation.invalid"></span>`validation.invalid` | {field} is not valid. |

## Admin API {#admin-api}

| Status | Code | Means |
| --- | --- | --- |
| 400 | <span id="admin_avatar_invalid"></span>`admin_avatar_invalid` | The avatar must be a full address starting with http:// or https://. |
| 409 | <span id="admin_email_taken"></span>`admin_email_taken` | An administrator with this email already exists. |
| 404 | <span id="admin_not_found"></span>`admin_not_found` | There is no such administrator. |
| 409 | <span id="admin_own_account"></span>`admin_own_account` | You cannot delete your own account. |
| 409 | <span id="admin_own_mfa"></span>`admin_own_mfa` | You cannot reset your own two-factor sign-in here. Manage it from your profile. |
| 400 | <span id="admin_password_required"></span>`admin_password_required` | A password is required. |
| 400 | <span id="admin_password_too_long"></span>`admin_password_too_long` | The password is too long. |
| 400 | <span id="admin_password_too_short"></span>`admin_password_too_short` | The password must be at least {min} characters. |
| 403 | <span id="admin_role_built_in"></span>`admin_role_built_in` | super_admin is built in and cannot be changed or removed. |
| 409 | <span id="admin_role_name_taken"></span>`admin_role_name_taken` | An admin role with this name already exists. |
| 404 | <span id="admin_role_not_found"></span>`admin_role_not_found` | There is no such admin role. |
| 409 | <span id="admin_session_current"></span>`admin_session_current` | That is the session you are using. Sign out to end it. |
| 404 | <span id="admin_session_not_found"></span>`admin_session_not_found` | There is no such session of yours. |
| 401 | <span id="admin_sign_in_code_wrong"></span>`admin_sign_in_code_wrong` | That code is not right. Check the time on your phone and try the newest code. |
| 401 | <span id="admin_sign_in_expired"></span>`admin_sign_in_expired` | Your sign-in has expired. Enter your password again. |
| 401 | <span id="admin_wrong_credentials"></span>`admin_wrong_credentials` | Wrong username or password. |
| 400 | <span id="admin_wrong_password"></span>`admin_wrong_password` | Your current password is not right. |
| 400 | <span id="api_identifier_fixed"></span>`api_identifier_fixed` | An API's identifier cannot be changed: register a new API instead. |
| 409 | <span id="api_identifier_taken"></span>`api_identifier_taken` | An API with this identifier already exists. |
| 404 | <span id="api_not_found"></span>`api_not_found` | There is no such API. |
| 400 | <span id="api_scope_foreign"></span>`api_scope_foreign` | {scope} is not a scope of this API. |
| 400 | <span id="api_too_many_scopes"></span>`api_too_many_scopes` | An API may have at most {max} scopes. |
| 400 | <span id="application_api_scope_foreign"></span>`application_api_scope_foreign` | One of the scopes is not a scope of {api}. |
| 400 | <span id="application_login_flow_unknown"></span>`application_login_flow_unknown` | There is no such login flow. |
| 400 | <span id="application_no_secret"></span>`application_no_secret` | A public client has no secret to rotate. |
| 404 | <span id="application_not_found"></span>`application_not_found` | There is no such application. |
| 400 | <span id="application_too_many_uris"></span>`application_too_many_uris` | An application may have at most {max} redirect URIs of each kind. |
| 400 | <span id="application_type_fixed"></span>`application_type_fixed` | An application's type cannot be changed: register a new application instead. |
| 400 | <span id="application_type_invalid"></span>`application_type_invalid` | The type must be one of: web, spa, native, m2m. |
| 413 | <span id="body_too_large"></span>`body_too_large` | The request is larger than this server accepts. |
| 400 | <span id="cache_cursor_invalid"></span>`cache_cursor_invalid` | The listing could not be continued. Start it again. |
| 404 | <span id="cache_database_not_found"></span>`cache_database_not_found` | There is no such Redis database. |
| 409 | <span id="cache_generation_key"></span>`cache_generation_key` | A generation counter cannot be removed. Clear its group instead. |
| 404 | <span id="cache_group_not_found"></span>`cache_group_not_found` | There is no such cache group. |
| 400 | <span id="cache_key_invalid"></span>`cache_key_invalid` | That is not a key name. |
| 409 | <span id="cache_key_not_editable"></span>`cache_key_not_editable` | Only a cached value in the cache database can be edited. |
| 404 | <span id="cache_key_not_found"></span>`cache_key_not_found` | That key is not in Redis any more. |
| 400 | <span id="cache_kind_invalid"></span>`cache_kind_invalid` | There is no such kind of key. |
| 400 | <span id="cache_ttl_invalid"></span>`cache_ttl_invalid` | The time to live has to be between 0 and {max} seconds. |
| 400 | <span id="cache_value_invalid"></span>`cache_value_invalid` | The value has to be valid JSON. |
| 403 | <span id="cross_origin"></span>`cross_origin` | Requests from {origin} are not allowed. |
| 403 | <span id="cross_site"></span>`cross_site` | Requests from other sites are not allowed. |
| 403 | <span id="forbidden"></span>`forbidden` | You do not have permission to do that. |
| 500 | <span id="internal"></span>`internal` | Something went wrong on our side. Try again in a moment. |
| 400 | <span id="invalid_body"></span>`invalid_body` | The request could not be read. Reload the page and try again. |
| 400 | <span id="language_base_protected"></span>`language_base_protected` | English is what every other language falls back to, so it stays. |
| 409 | <span id="language_code_taken"></span>`language_code_taken` | There is already a language with this code. |
| 400 | <span id="language_copy_missing"></span>`language_copy_missing` | There is no language {code} to copy. |
| 400 | <span id="language_default_protected"></span>`language_default_protected` | Make another language the default before removing this one. |
| 400 | <span id="language_default_required"></span>`language_default_required` | Make another language the default rather than unmarking this one. |
| 404 | <span id="language_not_found"></span>`language_not_found` | There is no such language. |
| 400 | <span id="login_flow_default_protected"></span>`login_flow_default_protected` | Make another flow the default before removing this one. |
| 400 | <span id="login_flow_default_required"></span>`login_flow_default_required` | Make another flow the default rather than unmarking this one. |
| 404 | <span id="login_flow_not_found"></span>`login_flow_not_found` | There is no such login flow. |
| 409 | <span id="login_flow_slug_taken"></span>`login_flow_slug_taken` | A flow with this identifier already exists. |
| 400 | <span id="logs_filter_invalid"></span>`logs_filter_invalid` | These filters cannot be applied. Check the dates and try again. |
| 400 | <span id="mail_content_key_unknown"></span>`mail_content_key_unknown` | {key} is not part of any email this server sends. |
| 502 | <span id="mail_test_failed"></span>`mail_test_failed` | The mail server did not take the message: {reason} |
| 400 | <span id="mail_test_needs_recipient"></span>`mail_test_needs_recipient` | Say where the test message should go. |
| 400 | <span id="mfa_code_wrong"></span>`mfa_code_wrong` | That code is not right. Check the time on your phone and try the newest code. |
| 400 | <span id="mfa_disabled"></span>`mfa_disabled` | Two-factor sign-in is not on. |
| 409 | <span id="mfa_enabled"></span>`mfa_enabled` | Two-factor sign-in is already on. Give a code from your current authenticator to replace it. |
| 400 | <span id="mfa_not_enrolling"></span>`mfa_not_enrolling` | Start setting up an authenticator first. |
| 409 | <span id="mfa_required"></span>`mfa_required` | Two-factor sign-in is required for every administrator and cannot be turned off. Replace your authenticator instead. |
| 404 | <span id="not_found"></span>`not_found` | There is nothing here. |
| 401 | <span id="not_signed_in"></span>`not_signed_in` | You are not signed in. |
| 429 | <span id="rate_limited"></span>`rate_limited` | Too many attempts. Try again in {seconds} seconds. |
| 503 | <span id="redis_not_configured"></span>`redis_not_configured` | This server runs without Redis, so there is no cache to look after. |
| 404 | <span id="session_not_found"></span>`session_not_found` | That session has ended or never existed. |
| 409 | <span id="setup_done"></span>`setup_done` | This panel has already been set up. |
| 400 | <span id="social_provider_kind_invalid"></span>`social_provider_kind_invalid` | The kind must be one of: {kinds}. |
| 404 | <span id="social_provider_not_found"></span>`social_provider_not_found` | There is no such provider. |
| 409 | <span id="social_provider_repointed"></span>`social_provider_repointed` | This provider already has linked accounts, so it cannot be pointed at a different one. Register a new provider instead. |
| 404 | <span id="social_provider_secret_missing"></span>`social_provider_secret_missing` | There is no secret stored for this provider. |
| 409 | <span id="social_provider_slug_taken"></span>`social_provider_slug_taken` | A provider with this identifier already exists. |
| 404 | <span id="sso_connection_not_found"></span>`sso_connection_not_found` | There is no such connection. |
| 400 | <span id="sso_discovery_failed"></span>`sso_discovery_failed` | The issuer's discovery document could not be read: {reason} |
| 400 | <span id="sso_domain_invalid"></span>`sso_domain_invalid` | {domain} is not a domain. |
| 409 | <span id="sso_domain_taken"></span>`sso_domain_taken` | {domain} already belongs to another connection. |
| 400 | <span id="sso_issuer_invalid"></span>`sso_issuer_invalid` | The issuer must be an https address. |
| 400 | <span id="sso_metadata_invalid"></span>`sso_metadata_invalid` | The SAML metadata could not be used: {reason} |
| 400 | <span id="sso_role_mapping_invalid"></span>`sso_role_mapping_invalid` | Every mapping needs a group and a role. |
| 400 | <span id="sso_role_unknown"></span>`sso_role_unknown` | A mapping names a role that does not exist. |
| 409 | <span id="sso_slug_taken"></span>`sso_slug_taken` | Another connection already uses the identifier {slug}. |
| 400 | <span id="sso_unreachable"></span>`sso_unreachable` | A connection without domains has to show its button on the sign-in page, and cannot be required: no address leads to it. |
| 403 | <span id="super_admin_only"></span>`super_admin_only` | Only a super admin can do that. |
| 409 | <span id="system_api"></span>`system_api` | This API is part of the server and cannot be deleted. |
| 403 | <span id="system_api_access"></span>`system_api_access` | Only a super admin can give an application access to the server's own APIs, or take it away. |
| 409 | <span id="system_application"></span>`system_application` | admin-cli is part of the server and cannot be deleted. Turn it off instead. |
| 403 | <span id="system_application_change"></span>`system_application_change` | Only a super admin can change an application that has access to the admin API, or rotate its secret. |
| 400 | <span id="token_preview_audience_unknown"></span>`token_preview_audience_unknown` | No API has that identifier. |
| 400 | <span id="token_preview_user_unknown"></span>`token_preview_user_unknown` | There is no such user to preview a token for. |
| 401 | <span id="token_refused"></span>`token_refused` | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 404 | <span id="translation_app_not_found"></span>`translation_app_not_found` | There is no such part of the product to translate. |
| 400 | <span id="translation_too_long"></span>`translation_too_long` | The text of {key} must be at most {max} characters. |
| 415 | <span id="unsupported_body"></span>`unsupported_body` | The request was sent in a form this server does not accept. |
| 409 | <span id="user_email_taken"></span>`user_email_taken` | A user with this email already exists. |
| 409 | <span id="user_field_name_taken"></span>`user_field_name_taken` | A field with this name already exists. |
| 404 | <span id="user_field_not_found"></span>`user_field_not_found` | There is no such field. |
| 404 | <span id="user_not_found"></span>`user_not_found` | There is no such user. |
| 400 | <span id="user_password_required"></span>`user_password_required` | A password is required. |
| 400 | <span id="user_password_too_short"></span>`user_password_too_short` | The password must be at least {min} characters. |
| 400 | <span id="user_role_application_unknown"></span>`user_role_application_unknown` | The role's application does not exist. |
| 409 | <span id="user_role_name_taken"></span>`user_role_name_taken` | A role with this name already exists in this scope. |
| 404 | <span id="user_role_not_found"></span>`user_role_not_found` | There is no such role. |
| 404 | <span id="user_role_not_held"></span>`user_role_not_held` | The user was not given that role directly. |
| 400 | <span id="user_role_scope_invalid"></span>`user_role_scope_invalid` | The scope must be "global" or "application". |
| 400 | <span id="user_role_unknown"></span>`user_role_unknown` | One of the roles does not exist. |
| 400 | <span id="user_roles_missing"></span>`user_roles_missing` | Name at least one role to assign. |
| 404 | <span id="user_social_account_not_found"></span>`user_social_account_not_found` | This user does not sign in with that. |
| 400 | <span id="validation.invalid"></span>`validation.invalid` | {field} is not valid. |

## OAuth errors {#oauth-errors}

The token, userinfo, revocation and introspection endpoints answer `{"error": "<code>", "error_description": "<sentence>"}` (RFC 6749 section 5.2). The authorization endpoint sends the same two back to the application's `redirect_uri`, with its `state` — once the redirect URI is known to be the application's. Before that, the browser is shown an error page instead, so an unregistered address never receives anything.

| Code | Status |
| --- | --- |
| `access_denied` | redirect |
| `insufficient_scope` | 403 |
| `invalid_client` | 401 |
| `invalid_grant` | 400 |
| `invalid_request` | 400 |
| `invalid_scope` | 400 |
| `invalid_token` | 401 |
| `login_required` | redirect |
| `server_error` | 500 |
| `unauthorized_client` | 400 |
| `unsupported_grant_type` | 400 |
| `unsupported_response_type` | redirect |
