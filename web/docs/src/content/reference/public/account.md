---
title: "Account"
description: "The API behind the hosted sign-in pages and a user's account page. Applications do not usually call it."
order: 4
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/account/organization {#get-api-v1-account-organization}

Describes the organisation the sign-in pages speak for.

Public, so it needs no session.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/organization"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/organization"))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/organization", nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(f"{ISSUER}/api/v1/account/organization")
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "organization": {
    "name": "Example",
    "logo_url": "https://example.com/logo.png",
    "support_email": "ada@example.com",
    "support_phone": "+1 555 0100",
    "terms_url": "https://example.com",
    "privacy_url": "https://example.com",
    "timezone": "Europe/London"
  }
}
```

## GET /api/v1/account/social-providers {#get-api-v1-account-social-providers}

Lists the sign-in buttons, with only what a button needs.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/social-providers"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/social-providers"))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/social-providers", nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(f"{ISSUER}/api/v1/account/social-providers")
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "providers": [
    {
      "slug": "example",
      "name": "Example",
      "kind": "string"
    }
  ]
}
```

## GET /api/v1/account/sso {#get-api-v1-account-sso}

Lists the organisations' identity providers the sign-in page offers a button for.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/sso"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/sso"))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/sso", nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(f"{ISSUER}/api/v1/account/sso")
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "connections": [
    {
      "slug": "example",
      "name": "Example",
      "protocol": "string"
    }
  ],
  "available": true
}
```

## POST /api/v1/account/sso/discover {#post-api-v1-account-sso-discover}

Says which connection an address signs in through, for "Sign in with SSO": the one that owns its domain, or no_sso_connection.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/sso/discover" \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/sso/discover"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "email": "ada@example.com"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"email": "ada@example.com"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/sso/discover", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/sso/discover",
    json={
        "email": "ada@example.com",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `email` <small>required</small> | string | An email address, at most 255 characters. |

#### Response

`200 OK`

```json
{
  "connection": {
    "slug": "example",
    "name": "Example",
    "protocol": "string"
  },
  "enforced": true
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 404 | [`no_sso_connection`](/reference/errors#no_sso_connection) | There is no single sign-on for that address. Sign in with your password instead. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## GET /api/v1/account/requests/:handle {#get-api-v1-account-requests-handle}

Describes a sign-in under way: the application it is for, as its sign-in page shows it.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/requests/$HANDLE"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/requests/" + handle))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/requests/"+handle, nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(f"{ISSUER}/api/v1/account/requests/{handle}")
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `handle` <small>required</small> | path |  |

#### Response

`200 OK`

```json
{
  "application": {
    "name": "Example",
    "logo_url": "https://example.com/logo.png",
    "website_url": "https://example.com",
    "privacy_url": "https://example.com",
    "terms_url": "https://example.com",
    "allow_registration": true
  },
  "login_hint": "string",
  "expires_at": "2026-09-28T09:30:00Z"
}
```

## GET /api/v1/account/login-options {#get-api-v1-account-login-options}

Returns the options of the login flow for the sign-in handle `request`, or the default flow.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/login-options?request=value"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/login-options" + "?request=value"))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/login-options"+"?request=value", nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ISSUER}/api/v1/account/login-options",
    params={
        "request": "value",
    },
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `request` | query |  |

#### Response

`200 OK`

```json
{
  "login": {
    "steps": [
      "consent"
    ],
    "allow_sign_in": true,
    "allow_registration": true,
    "allow_password_reset": true,
    "allow_remember_me": true,
    "allow_email_change": true
  }
}
```

## GET /api/v1/account/languages {#get-api-v1-account-languages}

Says which languages these pages may be shown in, and which one somebody gets before they have chosen.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/languages"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/languages"))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/languages", nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(f"{ISSUER}/api/v1/account/languages")
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "languages": [
    {
      "code": "string",
      "name": "Example",
      "native_name": "string"
    }
  ],
  "default": "string"
}
```

## GET /api/v1/account/languages/:code {#get-api-v1-account-languages-code}

Returns the sign-in pages' text in one offered language with every key filled in.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/languages/$CODE"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/languages/" + code))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/languages/"+code, nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(f"{ISSUER}/api/v1/account/languages/{code}")
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `code` <small>required</small> | path |  |

#### Response

`200 OK`

```json
{
  "language": {
    "code": "string",
    "name": "Example",
    "native_name": "string"
  },
  "messages": {}
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 404 | [`language_not_found`](/reference/errors#language_not_found) | There is no such language. |

## GET /api/v1/account/applications/:client_id {#get-api-v1-account-applications-client-id}

Describes an application by its client id, for the signed-out page to offer a way back to it.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/applications/$CLIENT_ID"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/applications/" + clientId))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/applications/"+clientID, nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(f"{ISSUER}/api/v1/account/applications/{client_id}")
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `client_id` <small>required</small> | path |  |

#### Response

`200 OK`

```json
{
  "application": {
    "name": "Example",
    "logo_url": "https://example.com/logo.png",
    "website_url": "https://example.com",
    "privacy_url": "https://example.com",
    "terms_url": "https://example.com",
    "allow_registration": true
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## POST /api/v1/account/login {#post-api-v1-account-login}

Signs a user in and, for a sign-in under way, says where to go next.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/login" \
  -H "Content-Type: application/json" \
  -d '{"request":"string","email":"ada@example.com","password":"correct horse battery staple","remember":true}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/login"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "request": "string",
                  "email": "ada@example.com",
                  "password": "correct horse battery staple",
                  "remember": true
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"request": "string",
	"email": "ada@example.com",
	"password": "correct horse battery staple",
	"remember": true
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/login", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/login",
    json={
        "request": "string",
        "email": "ada@example.com",
        "password": "correct horse battery staple",
        "remember": True,
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `request` | string | At most 64 characters. |
| `email` <small>required</small> | string | At most 255 characters. |
| `password` <small>required</small> | string | At most 128 characters. |
| `remember` | boolean | `remember` is the "stay signed in" box. A flow that does not offer it ignores whatever is sent. |

#### Response

`200 OK`

```json
{
  "redirect_to": "https://app.example.com/callback",
  "password_change_required": true,
  "reset_token": "eyJhbGciOiJSUzI1NiIs…",
  "code": {
    "handle": "string",
    "email": "ada@example.com",
    "expires_at": "2026-09-28T09:30:00Z",
    "resend_after": 1,
    "attempts_left": 1
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/account/login/code {#post-api-v1-account-login-code}

Finishes a sign-in that was waiting for an emailed code, and answers as the sign-in itself does: a cookie, and where to go next.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/login/code" \
  -H "Content-Type: application/json" \
  -d '{"handle":"string","code":"string"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/login/code"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "handle": "string",
                  "code": "string"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"handle": "string",
	"code": "string"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/login/code", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/login/code",
    json={
        "handle": "string",
        "code": "string",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `handle` <small>required</small> | string | At most 64 characters. |
| `code` <small>required</small> | string | At most 16 characters. |

#### Response

`200 OK`

```json
{
  "redirect_to": "https://app.example.com/callback",
  "password_change_required": true,
  "reset_token": "eyJhbGciOiJSUzI1NiIs…",
  "code": {
    "handle": "string",
    "email": "ada@example.com",
    "expires_at": "2026-09-28T09:30:00Z",
    "resend_after": 1,
    "attempts_left": 1
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/account/login/code/resend {#post-api-v1-account-login-code-resend}

Sends another code for a sign-in that is still waiting, and answers with the wait before the next one.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/login/code/resend" \
  -H "Content-Type: application/json" \
  -d '{"handle":"string"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/login/code/resend"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "handle": "string"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"handle": "string"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/login/code/resend", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/login/code/resend",
    json={
        "handle": "string",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `handle` <small>required</small> | string | At most 64 characters. |

#### Response

`200 OK`

```json
{
  "code": {
    "handle": "string",
    "email": "ada@example.com",
    "expires_at": "2026-09-28T09:30:00Z",
    "resend_after": 1,
    "attempts_left": 1
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/account/register {#post-api-v1-account-register}

Creates an account for a sign-in under way, and signs it in.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/register" \
  -H "Content-Type: application/json" \
  -d '{"request":"string","email":"ada@example.com","password":"correct horse battery staple"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/register"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "request": "string",
                  "email": "ada@example.com",
                  "password": "correct horse battery staple"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"request": "string",
	"email": "ada@example.com",
	"password": "correct horse battery staple"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/register", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/register",
    json={
        "request": "string",
        "email": "ada@example.com",
        "password": "correct horse battery staple",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `request` <small>required</small> | string | At most 64 characters. |
| `email` <small>required</small> | string | An email address, at most 255 characters. |
| `password` <small>required</small> | string | At most 72 characters. |
| `first_name` | string | At most 100 characters. |
| `last_name` | string | At most 100 characters. |
| `accept_terms` | boolean |  |
| `remember` | boolean |  |

#### Response

`200 OK`

```json
{
  "redirect_to": "https://app.example.com/callback",
  "password_change_required": true,
  "reset_token": "eyJhbGciOiJSUzI1NiIs…",
  "code": {
    "handle": "string",
    "email": "ada@example.com",
    "expires_at": "2026-09-28T09:30:00Z",
    "resend_after": 1,
    "attempts_left": 1
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`password_too_short`](/reference/errors#password_too_short) | The password must be at least {min} characters. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/account/forgot-password {#post-api-v1-account-forgot-password}

Sends a reset link.

It answers the same whether or not the address has an account.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/forgot-password" \
  -H "Content-Type: application/json" \
  -d '{"request":"string","email":"ada@example.com","language":"en"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/forgot-password"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "request": "string",
                  "email": "ada@example.com",
                  "language": "en"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"request": "string",
	"email": "ada@example.com",
	"language": "en"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/forgot-password", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/forgot-password",
    json={
        "request": "string",
        "email": "ada@example.com",
        "language": "en",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `request` | string | At most 64 characters. |
| `email` <small>required</small> | string | An email address, at most 255 characters. |
| `language` | string | `language` is the one the page was shown in, which the email is written in. One that is not offered, or none, is the default. At most 16 characters. |

#### Response

`202 Accepted`

```json
{
  "status": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## GET /api/v1/account/reset-password {#get-api-v1-account-reset-password}

Says whether a reset link still works.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ISSUER/api/v1/account/reset-password?token=eyJhbGciOiJSUzI1NiIs%E2%80%A6"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/reset-password" + "?token=eyJhbGciOiJSUzI1NiIs%E2%80%A6"))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/reset-password"+"?token=eyJhbGciOiJSUzI1NiIs%E2%80%A6", nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ISSUER}/api/v1/account/reset-password",
    params={
        "token": "eyJhbGciOiJSUzI1NiIs…",
    },
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `token` | query | The token the request is about. |

#### Response

`200 OK`

```json
{
  "valid": true
}
```

## POST /api/v1/account/reset-password {#post-api-v1-account-reset-password}

Sets a new password through a reset link.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/reset-password" \
  -H "Content-Type: application/json" \
  -d '{"token":"eyJhbGciOiJSUzI1NiIs…","password":"correct horse battery staple"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/reset-password"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "token": "eyJhbGciOiJSUzI1NiIs…",
                  "password": "correct horse battery staple"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"token": "eyJhbGciOiJSUzI1NiIs…",
	"password": "correct horse battery staple"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/reset-password", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/reset-password",
    json={
        "token": "eyJhbGciOiJSUzI1NiIs…",
        "password": "correct horse battery staple",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `token` <small>required</small> | string | At most 64 characters. |
| `password` <small>required</small> | string | At most 72 characters. |

#### Response

`200 OK`

```json
{
  "status": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`password_too_short`](/reference/errors#password_too_short) | The password must be at least {min} characters. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/account/verify-email {#post-api-v1-account-verify-email}

Uses a verification link.

It is a POST made when the person presses the button, so mail scanners that follow links cannot spend it.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/verify-email" \
  -H "Content-Type: application/json" \
  -d '{"token":"eyJhbGciOiJSUzI1NiIs…"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/verify-email"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "token": "eyJhbGciOiJSUzI1NiIs…"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"token": "eyJhbGciOiJSUzI1NiIs…"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/verify-email", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/verify-email",
    json={
        "token": "eyJhbGciOiJSUzI1NiIs…",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `token` <small>required</small> | string | At most 64 characters. |

#### Response

`200 OK`

```json
{
  "status": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/account/logout {#post-api-v1-account-logout}

Signs the user out of this server in this browser.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/logout"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/logout"))
        .POST(BodyPublishers.noBody())
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/logout", nil)
if err != nil {
	return err
}

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(f"{ISSUER}/api/v1/account/logout")
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "status": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |

## GET /api/v1/account/me {#get-api-v1-account-me}

Returns the signed-in user.

| | |
| --- | --- |
| Auth | User session, or a user's [Account API token](/guides/account-api) with `account.read` |

```bash title="cURL"
curl "$ISSUER/api/v1/account/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/me"))
        .header("Authorization", "Bearer " + ACCESS_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/me", nil)
if err != nil {
	return err
}
req.Header.Set("Authorization", "Bearer "+accessToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ISSUER}/api/v1/account/me",
    headers={"Authorization": f"Bearer {ACCESS_TOKEN}"},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "user": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "email": "ada@example.com",
    "email_verified": true,
    "first_name": "Ada",
    "last_name": "Lovelace",
    "created_at": "2026-09-28T09:30:00Z",
    "last_login_at": "2026-09-28T09:30:00Z"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`account_token_refused`](/reference/errors#account_token_refused) | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`account_scope_missing`](/reference/errors#account_scope_missing) | This access token needs the {scope} scope. |

## PATCH /api/v1/account/me {#patch-api-v1-account-me}

Changes the signed-in user's name.

| | |
| --- | --- |
| Auth | User session, or a user's [Account API token](/guides/account-api) with `account.write` |

```bash title="cURL"
curl -X PATCH "$ISSUER/api/v1/account/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Ada","last_name":"Lovelace"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/me"))
        .header("Authorization", "Bearer " + ACCESS_TOKEN)
        .header("Content-Type", "application/json")
        .method("PATCH", BodyPublishers.ofString("""
                {
                  "first_name": "Ada",
                  "last_name": "Lovelace"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"first_name": "Ada",
	"last_name": "Lovelace"
}`)
req, err := http.NewRequest(http.MethodPatch, issuer+"/api/v1/account/me", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+accessToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.patch(
    f"{ISSUER}/api/v1/account/me",
    json={
        "first_name": "Ada",
        "last_name": "Lovelace",
    },
    headers={"Authorization": f"Bearer {ACCESS_TOKEN}"},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `first_name` | string | At most 100 characters. |
| `last_name` | string | At most 100 characters. |

#### Response

`200 OK`

```json
{
  "user": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "email": "ada@example.com",
    "email_verified": true,
    "first_name": "Ada",
    "last_name": "Lovelace",
    "created_at": "2026-09-28T09:30:00Z",
    "last_login_at": "2026-09-28T09:30:00Z"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`account_token_refused`](/reference/errors#account_token_refused) | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`account_scope_missing`](/reference/errors#account_scope_missing) | This access token needs the {scope} scope. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |

## POST /api/v1/account/password {#post-api-v1-account-password}

Replaces the signed-in user's password.

| | |
| --- | --- |
| Auth | User session, or a user's [Account API token](/guides/account-api) with `account.write` |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/password" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"correct horse battery staple","new_password":"correct horse battery staple"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/password"))
        .header("Authorization", "Bearer " + ACCESS_TOKEN)
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "current_password": "correct horse battery staple",
                  "new_password": "correct horse battery staple"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"current_password": "correct horse battery staple",
	"new_password": "correct horse battery staple"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/password", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+accessToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/password",
    json={
        "current_password": "correct horse battery staple",
        "new_password": "correct horse battery staple",
    },
    headers={"Authorization": f"Bearer {ACCESS_TOKEN}"},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `current_password` <small>required</small> | string | At most 128 characters. |
| `new_password` <small>required</small> | string | At most 72 characters. |

#### Response

`200 OK`

```json
{
  "status": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`password_too_short`](/reference/errors#password_too_short) | The password must be at least {min} characters. |
| 401 | [`account_token_refused`](/reference/errors#account_token_refused) | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`account_scope_missing`](/reference/errors#account_scope_missing) | This access token needs the {scope} scope. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/account/email {#post-api-v1-account-email}

Starts moving the signed-in user to another address, given their current password.

It answers the same whether or not the address is taken, and only to a browser session: an access token is refused.

| | |
| --- | --- |
| Auth | User session, or a user's [Account API token](/guides/account-api) with `account.write` |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ISSUER/api/v1/account/email" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","current_password":"correct horse battery staple"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/email"))
        .header("Authorization", "Bearer " + ACCESS_TOKEN)
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "email": "ada@example.com",
                  "current_password": "correct horse battery staple"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"email": "ada@example.com",
	"current_password": "correct horse battery staple"
}`)
req, err := http.NewRequest(http.MethodPost, issuer+"/api/v1/account/email", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+accessToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ISSUER}/api/v1/account/email",
    json={
        "email": "ada@example.com",
        "current_password": "correct horse battery staple",
    },
    headers={"Authorization": f"Bearer {ACCESS_TOKEN}"},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `email` <small>required</small> | string | An email address, at most 255 characters. |
| `current_password` <small>required</small> | string | `current_password` proves the person at the session owns the account; a wrong one counts towards the lockout, as at sign-in. At most 128 characters. |

#### Response

`202 Accepted`

```json
{
  "status": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`account_token_refused`](/reference/errors#account_token_refused) | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`account_scope_missing`](/reference/errors#account_scope_missing) | This access token needs the {scope} scope. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## GET /api/v1/account/sessions {#get-api-v1-account-sessions}

Lists where the signed-in user is signed in.

| | |
| --- | --- |
| Auth | User session, or a user's [Account API token](/guides/account-api) with `account.read` |

```bash title="cURL"
curl "$ISSUER/api/v1/account/sessions" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/sessions"))
        .header("Authorization", "Bearer " + ACCESS_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/sessions", nil)
if err != nil {
	return err
}
req.Header.Set("Authorization", "Bearer "+accessToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ISSUER}/api/v1/account/sessions",
    headers={"Authorization": f"Bearer {ACCESS_TOKEN}"},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "sessions": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "ip": "string",
      "user_agent": "string",
      "signed_in_at": "2026-09-28T09:30:00Z",
      "expires_at": "2026-09-28T09:30:00Z",
      "current": true
    }
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`account_token_refused`](/reference/errors#account_token_refused) | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`account_scope_missing`](/reference/errors#account_scope_missing) | This access token needs the {scope} scope. |

## DELETE /api/v1/account/sessions/:id {#delete-api-v1-account-sessions-id}

Signs one of the user's other browsers out.

| | |
| --- | --- |
| Auth | User session, or a user's [Account API token](/guides/account-api) with `account.write` |

```bash title="cURL"
curl -X DELETE "$ISSUER/api/v1/account/sessions/$ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/sessions/" + id))
        .header("Authorization", "Bearer " + ACCESS_TOKEN)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, issuer+"/api/v1/account/sessions/"+id, nil)
if err != nil {
	return err
}
req.Header.Set("Authorization", "Bearer "+accessToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.delete(
    f"{ISSUER}/api/v1/account/sessions/{id}",
    headers={"Authorization": f"Bearer {ACCESS_TOKEN}"},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `id` <small>required</small> | path |  |

#### Response

`204 No Content`

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`account_token_refused`](/reference/errors#account_token_refused) | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`account_scope_missing`](/reference/errors#account_scope_missing) | This access token needs the {scope} scope. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |

## GET /api/v1/account/connected-applications {#get-api-v1-account-connected-applications}

Lists the applications that can still act for the user.

| | |
| --- | --- |
| Auth | User session, or a user's [Account API token](/guides/account-api) with `account.read` |

```bash title="cURL"
curl "$ISSUER/api/v1/account/connected-applications" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/connected-applications"))
        .header("Authorization", "Bearer " + ACCESS_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/connected-applications", nil)
if err != nil {
	return err
}
req.Header.Set("Authorization", "Bearer "+accessToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ISSUER}/api/v1/account/connected-applications",
    headers={"Authorization": f"Bearer {ACCESS_TOKEN}"},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "applications": [
    {
      "client_id": "shop-web",
      "name": "Example",
      "logo_url": "https://example.com/logo.png",
      "website_url": "https://example.com",
      "scopes": [
        "openid profile email"
      ],
      "authorized_at": "2026-09-28T09:30:00Z",
      "last_used_at": "2026-09-28T09:30:00Z"
    }
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`account_token_refused`](/reference/errors#account_token_refused) | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`account_scope_missing`](/reference/errors#account_scope_missing) | This access token needs the {scope} scope. |

## DELETE /api/v1/account/connected-applications/:client_id {#delete-api-v1-account-connected-applications-client-id}

Revokes an application's refresh tokens for the user.

| | |
| --- | --- |
| Auth | User session, or a user's [Account API token](/guides/account-api) with `account.write` |

```bash title="cURL"
curl -X DELETE "$ISSUER/api/v1/account/connected-applications/$CLIENT_ID" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ISSUER + "/api/v1/account/connected-applications/" + clientId))
        .header("Authorization", "Bearer " + ACCESS_TOKEN)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, issuer+"/api/v1/account/connected-applications/"+clientID, nil)
if err != nil {
	return err
}
req.Header.Set("Authorization", "Bearer "+accessToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.delete(
    f"{ISSUER}/api/v1/account/connected-applications/{client_id}",
    headers={"Authorization": f"Bearer {ACCESS_TOKEN}"},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `client_id` <small>required</small> | path |  |

#### Response

`204 No Content`

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`account_token_refused`](/reference/errors#account_token_refused) | This access token is not accepted by the account API: it has expired, is for another API, or its user can no longer sign in. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`account_scope_missing`](/reference/errors#account_scope_missing) | This access token needs the {scope} scope. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
