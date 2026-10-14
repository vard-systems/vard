---
title: "Administrators"
description: "The panel's administrators."
order: 18
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/security {#get-api-v1-admin-security}

Returns the administrators' sign-in settings and how many have an authenticator.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/security" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/security"))
        .header("Cookie", "loginer_session=" + SESSION)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/security", nil)
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
    f"{ADMIN_API}/api/v1/admin/security",
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "require_mfa": true,
  "administrators": 1,
  "with_mfa": 1
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## PATCH /api/v1/admin/security {#patch-api-v1-admin-security}

Changes those settings.

Requiring a second factor applies on each administrator's next page load.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/security" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"require_mfa":true}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/security"))
        .header("Cookie", "loginer_session=" + SESSION)
        .header("Content-Type", "application/json")
        .method("PATCH", BodyPublishers.ofString("""
                {
                  "require_mfa": true
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"require_mfa": true
}`)
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/security", body)
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
    f"{ADMIN_API}/api/v1/admin/security",
    json={
        "require_mfa": True,
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `require_mfa` | boolean or null |  |

#### Response

`200 OK`

```json
{
  "require_mfa": true,
  "administrators": 1,
  "with_mfa": 1
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## GET /api/v1/admin/admins {#get-api-v1-admin-admins}

Returns a page of administrators, newest first.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/admins?search=value&status=value&role=value" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admins" + "?search=value&status=value&role=value"))
        .header("Cookie", "loginer_session=" + SESSION)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/admins"+"?search=value&status=value&role=value", nil)
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
    f"{ADMIN_API}/api/v1/admin/admins",
    params={
        "search": "value",
        "status": "value",
        "role": "value",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `search` | query |  |
| `status` | query |  |
| `role` | query |  |

#### Response

`200 OK`

```json
{
  "admins": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "username": "string",
      "email": "ada@example.com",
      "first_name": "Ada",
      "last_name": "Lovelace",
      "full_name": "string",
      "status": "string",
      "assignments": [
        {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "role": {
            "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
            "name": "Example"
          },
          "application": {
            "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
            "name": "Example"
          }
        }
      ],
      "permissions": [
        "string"
      ],
      "scoped_permissions": {},
      "is_super_admin": true,
      "mfa_enabled": true,
      "last_login_at": "2026-09-28T09:30:00Z",
      "last_login_ip": "string",
      "created_at": "2026-09-28T09:30:00Z",
      "updated_at": "2026-09-28T09:30:00Z"
    }
  ],
  "total": 1,
  "limit": 1,
  "offset": 1
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## POST /api/v1/admin/admins {#post-api-v1-admin-admins}

Adds an administrator, with the password and roles the super admin chose for them.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/admins" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","first_name":"Ada","status":"active"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admins"))
        .header("Cookie", "loginer_session=" + SESSION)
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "email": "ada@example.com",
                  "first_name": "Ada",
                  "status": "active"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"email": "ada@example.com",
	"first_name": "Ada",
	"status": "active"
}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/admins", body)
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
    f"{ADMIN_API}/api/v1/admin/admins",
    json={
        "email": "ada@example.com",
        "first_name": "Ada",
        "status": "active",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `email` <small>required</small> | string | An email address, at most 255 characters. |
| `first_name` <small>required</small> | string | At most 100 characters. |
| `last_name` | string | At most 100 characters. |
| `status` <small>required</small> | one of `active`, `suspended`, `disabled` | `status` says whether the account may sign in. Only an active one may. |
| `password` | string | `password` is required when creating an administrator. When updating one it is optional: left empty, the password they have is kept. At most 72 characters. |
| `confirm_password` | string | The same as `password`. |
| `assignments` | array of AssignmentRequest |  |
| `assignments[].role_id` | string (uuid) |  |
| `assignments[].application_id` | string (uuid) or null |  |

#### Response

Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{"error", "error_description"}`.

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`admin_password_required`](/reference/errors#admin_password_required) | A password is required. |
| 400 | [`admin_password_too_short`](/reference/errors#admin_password_too_short) | The password must be at least {min} characters. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 409 | [`admin_email_taken`](/reference/errors#admin_email_taken) | An administrator with this email already exists. |

## GET /api/v1/admin/admins/:id {#get-api-v1-admin-admins-id}

Returns one administrator.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/admins/$ID" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admins/" + id))
        .header("Cookie", "loginer_session=" + SESSION)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/admins/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/admins/{id}",
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `id` <small>required</small> | path |  |

#### Response

`200 OK`

```json
{
  "admin": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "username": "string",
    "email": "ada@example.com",
    "first_name": "Ada",
    "last_name": "Lovelace",
    "full_name": "string",
    "status": "string",
    "assignments": [
      {
        "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
        "role": {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example"
        },
        "application": {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example"
        }
      }
    ],
    "permissions": [
      "string"
    ],
    "scoped_permissions": {},
    "is_super_admin": true,
    "mfa_enabled": true,
    "last_login_at": "2026-09-28T09:30:00Z",
    "last_login_ip": "string",
    "created_at": "2026-09-28T09:30:00Z",
    "updated_at": "2026-09-28T09:30:00Z"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`admin_not_found`](/reference/errors#admin_not_found) | There is no such administrator. |

## PATCH /api/v1/admin/admins/:id {#patch-api-v1-admin-admins-id}

Replaces an administrator's details, status, roles and optionally password.

A new password, or losing the right to sign in, ends their sessions.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/admins/$ID" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","first_name":"Ada","status":"active"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admins/" + id))
        .header("Cookie", "loginer_session=" + SESSION)
        .header("Content-Type", "application/json")
        .method("PATCH", BodyPublishers.ofString("""
                {
                  "email": "ada@example.com",
                  "first_name": "Ada",
                  "status": "active"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"email": "ada@example.com",
	"first_name": "Ada",
	"status": "active"
}`)
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/admins/"+id, body)
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
    f"{ADMIN_API}/api/v1/admin/admins/{id}",
    json={
        "email": "ada@example.com",
        "first_name": "Ada",
        "status": "active",
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `id` <small>required</small> | path |  |

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `email` <small>required</small> | string | An email address, at most 255 characters. |
| `first_name` <small>required</small> | string | At most 100 characters. |
| `last_name` | string | At most 100 characters. |
| `status` <small>required</small> | one of `active`, `suspended`, `disabled` | `status` says whether the account may sign in. Only an active one may. |
| `password` | string | `password` is required when creating an administrator. When updating one it is optional: left empty, the password they have is kept. At most 72 characters. |
| `confirm_password` | string | The same as `password`. |
| `assignments` | array of AssignmentRequest |  |
| `assignments[].role_id` | string (uuid) |  |
| `assignments[].application_id` | string (uuid) or null |  |

#### Response

Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{"error", "error_description"}`.

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`admin_password_required`](/reference/errors#admin_password_required) | A password is required. |
| 400 | [`admin_password_too_short`](/reference/errors#admin_password_too_short) | The password must be at least {min} characters. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`admin_not_found`](/reference/errors#admin_not_found) | There is no such administrator. |
| 409 | [`admin_email_taken`](/reference/errors#admin_email_taken) | An administrator with this email already exists. |

## DELETE /api/v1/admin/admins/:id {#delete-api-v1-admin-admins-id}

Removes an administrator for good.

Nobody can remove themselves: that is how a panel ends up with no one able to manage it.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/admins/$ID" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admins/" + id))
        .header("Cookie", "loginer_session=" + SESSION)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/admins/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/admins/{id}",
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
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`admin_not_found`](/reference/errors#admin_not_found) | There is no such administrator. |
| 409 | [`admin_own_account`](/reference/errors#admin_own_account) | You cannot delete your own account. |

## DELETE /api/v1/admin/admins/:id/mfa {#delete-api-v1-admin-admins-id-mfa}

Removes another administrator's second factor and signs them out everywhere.

Your own is managed from your profile with a code.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/admins/$ID/mfa" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admins/" + id + "/mfa"))
        .header("Cookie", "loginer_session=" + SESSION)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/admins/"+id+"/mfa", nil)
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
    f"{ADMIN_API}/api/v1/admin/admins/{id}/mfa",
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `id` <small>required</small> | path |  |

#### Response

Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{"error", "error_description"}`.

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`admin_not_found`](/reference/errors#admin_not_found) | There is no such administrator. |
| 409 | [`admin_own_mfa`](/reference/errors#admin_own_mfa) | You cannot reset your own two-factor sign-in here. Manage it from your profile. |
