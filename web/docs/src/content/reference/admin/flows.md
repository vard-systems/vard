---
title: "Login flows"
description: "The steps a sign-in goes through, per application."
order: 13
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/login-flows {#get-api-v1-admin-login-flows}

Returns every flow, the steps one can be made of, and how many applications each flow signs people in for.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`login_flows.read`](/reference/permissions#login_flows-read) |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/login-flows" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/login-flows"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/login-flows", nil)
if err != nil {
	return err
}
req.Header.Set("Authorization", "Bearer "+adminToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ADMIN_API}/api/v1/admin/login-flows",
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "flows": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "name": "Example",
      "slug": "example",
      "description": "string",
      "is_default": true,
      "is_enabled": true,
      "steps": [
        "consent"
      ],
      "allow_sign_in": true,
      "allow_registration": true,
      "allow_password_reset": true,
      "allow_remember_me": true,
      "verify_email_on_register": true,
      "require_verified_email": true,
      "allow_email_change": true,
      "session_lifetime_hours": 3600,
      "applications": 1,
      "planned": [
        "consent"
      ],
      "created_at": "2026-09-28T09:30:00Z",
      "updated_at": "2026-09-28T09:30:00Z"
    }
  ],
  "step_kinds": [
    {
      "step": "consent",
      "label": "string",
      "description": "string",
      "fixed": true,
      "implemented": true
    }
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## GET /api/v1/admin/login-flows/:id {#get-api-v1-admin-login-flows-id}

Returns one flow.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`login_flows.read`](/reference/permissions#login_flows-read) |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/login-flows/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/login-flows/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/login-flows/"+id, nil)
if err != nil {
	return err
}
req.Header.Set("Authorization", "Bearer "+adminToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.get(
    f"{ADMIN_API}/api/v1/admin/login-flows/{id}",
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
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
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`login_flow_not_found`](/reference/errors#login_flow_not_found) | There is no such login flow. |

## POST /api/v1/admin/login-flows {#post-api-v1-admin-login-flows}

Adds a flow, disabled until an administrator enables it.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`login_flows.write`](/reference/permissions#login_flows-write) |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/login-flows" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/login-flows"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {}
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/login-flows", body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+adminToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.post(
    f"{ADMIN_API}/api/v1/admin/login-flows",
    json={},
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `name` | string or null |  |
| `slug` | string or null |  |
| `description` | string or null |  |
| `is_default` | boolean or null |  |
| `is_enabled` | boolean or null |  |
| `steps` | array of one of `consent`, `email_code`, `identifier`, `password`, `social`, `terms`, `totp` or null |  |
| `allow_sign_in` | boolean or null |  |
| `allow_registration` | boolean or null |  |
| `allow_password_reset` | boolean or null |  |
| `allow_remember_me` | boolean or null |  |
| `verify_email_on_register` | boolean or null |  |
| `require_verified_email` | boolean or null |  |
| `allow_email_change` | boolean or null |  |
| `session_lifetime_hours` | integer or null |  |

#### Response

Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{"error", "error_description"}`.

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 409 | [`login_flow_slug_taken`](/reference/errors#login_flow_slug_taken) | A flow with this identifier already exists. |

## PATCH /api/v1/admin/login-flows/:id {#patch-api-v1-admin-login-flows-id}

Changes a flow.

Its identifier is not among them: an application and an export both name a flow by it.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`login_flows.write`](/reference/permissions#login_flows-write) |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/login-flows/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/login-flows/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .header("Content-Type", "application/json")
        .method("PATCH", BodyPublishers.ofString("""
                {}
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{}`)
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/login-flows/"+id, body)
if err != nil {
	return err
}
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+adminToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.patch(
    f"{ADMIN_API}/api/v1/admin/login-flows/{id}",
    json={},
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
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
| `name` | string or null |  |
| `slug` | string or null |  |
| `description` | string or null |  |
| `is_default` | boolean or null |  |
| `is_enabled` | boolean or null |  |
| `steps` | array of one of `consent`, `email_code`, `identifier`, `password`, `social`, `terms`, `totp` or null |  |
| `allow_sign_in` | boolean or null |  |
| `allow_registration` | boolean or null |  |
| `allow_password_reset` | boolean or null |  |
| `allow_remember_me` | boolean or null |  |
| `verify_email_on_register` | boolean or null |  |
| `require_verified_email` | boolean or null |  |
| `allow_email_change` | boolean or null |  |
| `session_lifetime_hours` | integer or null |  |

#### Response

Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{"error", "error_description"}`.

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`login_flow_default_required`](/reference/errors#login_flow_default_required) | Make another flow the default rather than unmarking this one. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`login_flow_not_found`](/reference/errors#login_flow_not_found) | There is no such login flow. |

## DELETE /api/v1/admin/login-flows/:id {#delete-api-v1-admin-login-flows-id}

Removes a flow.

The applications signing people in with it fall back to the default, which is why the default itself cannot go.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`login_flows.write`](/reference/permissions#login_flows-write) |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/login-flows/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/login-flows/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/login-flows/"+id, nil)
if err != nil {
	return err
}
req.Header.Set("Authorization", "Bearer "+adminToken)

res, err := http.DefaultClient.Do(req)
if err != nil {
	return err
}
defer res.Body.Close()
```

```python title="Python"
response = requests.delete(
    f"{ADMIN_API}/api/v1/admin/login-flows/{id}",
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
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
| 400 | [`login_flow_default_protected`](/reference/errors#login_flow_default_protected) | Make another flow the default before removing this one. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`login_flow_not_found`](/reference/errors#login_flow_not_found) | There is no such login flow. |
