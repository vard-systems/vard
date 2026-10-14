---
title: "Second factor"
description: "An administrator's authenticator app and recovery codes."
order: 4
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/mfa {#get-api-v1-admin-mfa}

Describes the administrator's second factor.

| | |
| --- | --- |
| Auth | Admin session, including one still setting up a second factor |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/mfa" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/mfa"))
        .header("Cookie", "loginer_session=" + SESSION)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/mfa", nil)
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
    f"{ADMIN_API}/api/v1/admin/mfa",
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "mfa": {
    "enabled": true,
    "required": true,
    "confirmed_at": "2026-09-28T09:30:00Z",
    "last_used_at": "2026-09-28T09:30:00Z",
    "recovery_codes_left": 1
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |

## POST /api/v1/admin/mfa/totp {#post-api-v1-admin-mfa-totp}

Starts setting up an authenticator app, and answers the secret and the otpauth URI to show as a QR code.

| | |
| --- | --- |
| Auth | Admin session, including one still setting up a second factor |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/mfa/totp" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"code":"string"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/mfa/totp"))
        .header("Cookie", "loginer_session=" + SESSION)
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
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/mfa/totp", body)
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
    f"{ADMIN_API}/api/v1/admin/mfa/totp",
    json={
        "code": "string",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `code` | string |  |

#### Response

`200 OK`

```json
{
  "enrolment": {
    "secret": "string",
    "uri": "https://example.com"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`mfa_code_wrong`](/reference/errors#mfa_code_wrong) | That code is not right. Check the time on your phone and try the newest code. |
| 400 | [`mfa_disabled`](/reference/errors#mfa_disabled) | Two-factor sign-in is not on. |
| 400 | [`mfa_not_enrolling`](/reference/errors#mfa_not_enrolling) | Start setting up an authenticator first. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 409 | [`mfa_enabled`](/reference/errors#mfa_enabled) | Two-factor sign-in is already on. Give a code from your current authenticator to replace it. |
| 409 | [`mfa_required`](/reference/errors#mfa_required) | Two-factor sign-in is required for every administrator and cannot be turned off. Replace your authenticator instead. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/admin/mfa/totp/confirm {#post-api-v1-admin-mfa-totp-confirm}

Finishes setting up with a code from the app, and answers the recovery codes — the only time they are shown.

| | |
| --- | --- |
| Auth | Admin session, including one still setting up a second factor |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/mfa/totp/confirm" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"code":"string"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/mfa/totp/confirm"))
        .header("Cookie", "loginer_session=" + SESSION)
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
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/mfa/totp/confirm", body)
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
    f"{ADMIN_API}/api/v1/admin/mfa/totp/confirm",
    json={
        "code": "string",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `code` | string |  |

#### Response

`200 OK`

```json
{
  "recovery_codes": [
    "string"
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`mfa_code_wrong`](/reference/errors#mfa_code_wrong) | That code is not right. Check the time on your phone and try the newest code. |
| 400 | [`mfa_disabled`](/reference/errors#mfa_disabled) | Two-factor sign-in is not on. |
| 400 | [`mfa_not_enrolling`](/reference/errors#mfa_not_enrolling) | Start setting up an authenticator first. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 409 | [`mfa_enabled`](/reference/errors#mfa_enabled) | Two-factor sign-in is already on. Give a code from your current authenticator to replace it. |
| 409 | [`mfa_required`](/reference/errors#mfa_required) | Two-factor sign-in is required for every administrator and cannot be turned off. Replace your authenticator instead. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## DELETE /api/v1/admin/mfa/totp {#delete-api-v1-admin-mfa-totp}

Turns two-factor sign-in off.

| | |
| --- | --- |
| Auth | Admin session |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/mfa/totp" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"code":"string"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/mfa/totp"))
        .header("Cookie", "loginer_session=" + SESSION)
        .header("Content-Type", "application/json")
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"code": "string"
}`)
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/mfa/totp", body)
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
response = requests.delete(
    f"{ADMIN_API}/api/v1/admin/mfa/totp",
    json={
        "code": "string",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `code` | string |  |

#### Response

`204 No Content`

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`mfa_code_wrong`](/reference/errors#mfa_code_wrong) | That code is not right. Check the time on your phone and try the newest code. |
| 400 | [`mfa_disabled`](/reference/errors#mfa_disabled) | Two-factor sign-in is not on. |
| 400 | [`mfa_not_enrolling`](/reference/errors#mfa_not_enrolling) | Start setting up an authenticator first. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 409 | [`mfa_enabled`](/reference/errors#mfa_enabled) | Two-factor sign-in is already on. Give a code from your current authenticator to replace it. |
| 409 | [`mfa_required`](/reference/errors#mfa_required) | Two-factor sign-in is required for every administrator and cannot be turned off. Replace your authenticator instead. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |

## POST /api/v1/admin/mfa/recovery-codes {#post-api-v1-admin-mfa-recovery-codes}

Replaces the recovery codes.

| | |
| --- | --- |
| Auth | Admin session |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/mfa/recovery-codes" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"code":"string"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/mfa/recovery-codes"))
        .header("Cookie", "loginer_session=" + SESSION)
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
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/mfa/recovery-codes", body)
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
    f"{ADMIN_API}/api/v1/admin/mfa/recovery-codes",
    json={
        "code": "string",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `code` | string |  |

#### Response

`200 OK`

```json
{
  "recovery_codes": [
    "string"
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`mfa_code_wrong`](/reference/errors#mfa_code_wrong) | That code is not right. Check the time on your phone and try the newest code. |
| 400 | [`mfa_disabled`](/reference/errors#mfa_disabled) | Two-factor sign-in is not on. |
| 400 | [`mfa_not_enrolling`](/reference/errors#mfa_not_enrolling) | Start setting up an authenticator first. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 409 | [`mfa_enabled`](/reference/errors#mfa_enabled) | Two-factor sign-in is already on. Give a code from your current authenticator to replace it. |
| 409 | [`mfa_required`](/reference/errors#mfa_required) | Two-factor sign-in is required for every administrator and cannot be turned off. Replace your authenticator instead. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |
