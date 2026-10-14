---
title: "Signing in"
description: "Signing an administrator in and out, and who is signed in."
order: 3
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## POST /api/v1/admin/auth/login {#post-api-v1-admin-auth-login}

Checks the credentials and sets the session cookie.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"string","password":"correct horse battery staple"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/auth/login"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "username": "string",
                  "password": "correct horse battery staple"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"username": "string",
	"password": "correct horse battery staple"
}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/auth/login", body)
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
    f"{ADMIN_API}/api/v1/admin/auth/login",
    json={
        "username": "string",
        "password": "correct horse battery staple",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `username` <small>required</small> | string |  |
| `password` <small>required</small> | string |  |

#### Response

`200 OK`

```json
{
  "next": "enroll"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`admin_wrong_credentials`](/reference/errors#admin_wrong_credentials) | Wrong username or password. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## GET /api/v1/admin/auth/session {#get-api-v1-admin-auth-session}

Reports how far this browser's session has got (password, code, enrolment).

It needs no session.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/auth/session"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/auth/session"))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/auth/session", nil)
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
response = requests.get(f"{ADMIN_API}/api/v1/admin/auth/session")
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "state": "enroll",
  "mfa_required": true
}
```

## POST /api/v1/admin/auth/mfa {#post-api-v1-admin-auth-mfa}

Finishes a sign-in waiting for a second factor.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/auth/mfa" \
  -H "Content-Type: application/json" \
  -d '{"code":"string"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/auth/mfa"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "code": "string"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"code": "string"
}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/auth/mfa", body)
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
    f"{ADMIN_API}/api/v1/admin/auth/mfa",
    json={
        "code": "string",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `code` <small>required</small> | string | At most 32 characters. |

#### Response

`200 OK`

```json
{
  "admin": {
    "id": "string",
    "username": "string",
    "email": "ada@example.com",
    "full_name": "string",
    "first_name": "Ada",
    "last_name": "Lovelace",
    "avatar_url": "https://example.com",
    "status": "string",
    "last_login_at": "2026-09-28T09:30:00Z",
    "roles": [
      "string"
    ],
    "permissions": [
      "string"
    ],
    "scoped_permissions": {},
    "is_super_admin": true,
    "mfa_enabled": true
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`admin_sign_in_code_wrong`](/reference/errors#admin_sign_in_code_wrong) | That code is not right. Check the time on your phone and try the newest code. |
| 401 | [`admin_sign_in_expired`](/reference/errors#admin_sign_in_expired) | Your sign-in has expired. Enter your password again. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/admin/auth/logout {#post-api-v1-admin-auth-logout}

Revokes the session and clears the cookie.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/auth/logout"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/auth/logout"))
        .POST(BodyPublishers.noBody())
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/auth/logout", nil)
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
response = requests.post(f"{ADMIN_API}/api/v1/admin/auth/logout")
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

## GET /api/v1/admin/me {#get-api-v1-admin-me}

Returns the signed-in administrator, and tells the browser whether its session is still alive.

The organisation's public name and logo come with it for the panel's header.

| | |
| --- | --- |
| Auth | Admin session |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/me" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/me"))
        .header("Cookie", "loginer_session=" + SESSION)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/me", nil)
if err != nil {
	return err
}
req.AddCookie(&http.Cookie{Name: "loginer_session", Value: session})

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ADMIN_API}/api/v1/admin/me",
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "admin": {
    "id": "string",
    "username": "string",
    "email": "ada@example.com",
    "full_name": "string",
    "first_name": "Ada",
    "last_name": "Lovelace",
    "avatar_url": "https://example.com",
    "status": "string",
    "last_login_at": "2026-09-28T09:30:00Z",
    "roles": [
      "string"
    ],
    "permissions": [
      "string"
    ],
    "scoped_permissions": {},
    "is_super_admin": true,
    "mfa_enabled": true
  },
  "organization": {
    "name": "Example",
    "logo_url": "https://example.com/logo.png"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |

## PATCH /api/v1/admin/me {#patch-api-v1-admin-me}

Changes the caller's own name and address, nothing else, so every administrator may use it.

| | |
| --- | --- |
| Auth | Admin session |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/me" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Ada","email":"ada@example.com"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/me"))
        .header("Cookie", "loginer_session=" + SESSION)
        .header("Content-Type", "application/json")
        .method("PATCH", BodyPublishers.ofString("""
                {
                  "first_name": "Ada",
                  "email": "ada@example.com"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"first_name": "Ada",
	"email": "ada@example.com"
}`)
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/me", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")
req.AddCookie(&http.Cookie{Name: "loginer_session", Value: session})

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.patch(
    f"{ADMIN_API}/api/v1/admin/me",
    json={
        "first_name": "Ada",
        "email": "ada@example.com",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `first_name` <small>required</small> | string | At most 100 characters. |
| `last_name` | string | At most 100 characters. |
| `email` <small>required</small> | string | An email address, at most 255 characters. |
| `avatar_url` | string |  |
| `current_password` | string |  |

#### Response

`200 OK`

```json
{
  "admin": {
    "id": "string",
    "username": "string",
    "email": "ada@example.com",
    "full_name": "string",
    "first_name": "Ada",
    "last_name": "Lovelace",
    "avatar_url": "https://example.com",
    "status": "string",
    "last_login_at": "2026-09-28T09:30:00Z",
    "roles": [
      "string"
    ],
    "permissions": [
      "string"
    ],
    "scoped_permissions": {},
    "is_super_admin": true,
    "mfa_enabled": true
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`admin_avatar_invalid`](/reference/errors#admin_avatar_invalid) | The avatar must be a full address starting with http:// or https://. |
| 400 | [`admin_wrong_password`](/reference/errors#admin_wrong_password) | Your current password is not right. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 409 | [`admin_email_taken`](/reference/errors#admin_email_taken) | An administrator with this email already exists. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/admin/me/password {#post-api-v1-admin-me-password}

Sets the caller's own password, given the one they have.

Every other session they have open ends; the one they are using stays.

| | |
| --- | --- |
| Auth | Admin session |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/me/password" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"correct horse battery staple","new_password":"correct horse battery staple"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/me/password"))
        .header("Cookie", "loginer_session=" + SESSION)
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
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/me/password", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")
req.AddCookie(&http.Cookie{Name: "loginer_session", Value: session})

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ADMIN_API}/api/v1/admin/me/password",
    json={
        "current_password": "correct horse battery staple",
        "new_password": "correct horse battery staple",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `current_password` <small>required</small> | string |  |
| `new_password` <small>required</small> | string |  |

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
| 400 | [`admin_password_too_long`](/reference/errors#admin_password_too_long) | The password is too long. |
| 400 | [`admin_password_too_short`](/reference/errors#admin_password_too_short) | The password must be at least {min} characters. |
| 400 | [`admin_wrong_password`](/reference/errors#admin_wrong_password) | Your current password is not right. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## GET /api/v1/admin/sessions {#get-api-v1-admin-sessions}

Lists the caller's own sessions, so they can see where they are signed in.

| | |
| --- | --- |
| Auth | Admin session |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/sessions" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/sessions"))
        .header("Cookie", "loginer_session=" + SESSION)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/sessions", nil)
if err != nil {
	return err
}
req.AddCookie(&http.Cookie{Name: "loginer_session", Value: session})

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ADMIN_API}/api/v1/admin/sessions",
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "sessions": [
    {
      "id": "string",
      "ip": "string",
      "user_agent": "string",
      "created_at": "2026-09-28T09:30:00Z",
      "expires_at": "2026-09-28T09:30:00Z",
      "active": true,
      "current": true
    }
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |

## DELETE /api/v1/admin/sessions/:id {#delete-api-v1-admin-sessions-id}

Signs one of the administrator's other browsers out.

Another administrator's session is answered as not found.

| | |
| --- | --- |
| Auth | Admin session |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/sessions/$ID" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/sessions/" + id))
        .header("Cookie", "loginer_session=" + SESSION)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/sessions/"+id, nil)
if err != nil {
	return err
}
req.AddCookie(&http.Cookie{Name: "loginer_session", Value: session})

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.delete(
    f"{ADMIN_API}/api/v1/admin/sessions/{id}",
    cookies={"loginer_session": SESSION},
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
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 404 | [`admin_session_not_found`](/reference/errors#admin_session_not_found) | There is no such session of yours. |
| 409 | [`admin_session_current`](/reference/errors#admin_session_current) | That is the session you are using. Sign out to end it. |

## DELETE /api/v1/admin/sessions {#delete-api-v1-admin-sessions}

Signs the administrator out everywhere but here: what to do after using a shared computer, or losing a laptop.

| | |
| --- | --- |
| Auth | Admin session |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/sessions" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/sessions"))
        .header("Cookie", "loginer_session=" + SESSION)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/sessions", nil)
if err != nil {
	return err
}
req.AddCookie(&http.Cookie{Name: "loginer_session", Value: session})

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.delete(
    f"{ADMIN_API}/api/v1/admin/sessions",
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "ended": 1
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
