---
title: "Applications"
description: "The applications that sign users in: their settings, secrets and API access."
order: 15
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/applications {#get-api-v1-admin-applications}

Returns a page of the applications the administrator can see, sorted by name, each with how many roles it defines.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/applications?search=value&type=value" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications" + "?search=value&type=value"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/applications"+"?search=value&type=value", nil)
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
    f"{ADMIN_API}/api/v1/admin/applications",
    params={
        "search": "value",
        "type": "value",
    },
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `search` | query |  |
| `type` | query |  |

#### Response

`200 OK`

```json
{
  "applications": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "name": "Example",
      "description": "string",
      "type": "m2m",
      "logo_url": "https://example.com/logo.png",
      "website_url": "https://example.com",
      "privacy_url": "https://example.com",
      "terms_url": "https://example.com",
      "client_id": "shop-web",
      "client_id_issued_at": 1790000000,
      "has_secret": true,
      "secret_hint": "string",
      "secret_created_at": "2026-09-28T09:30:00Z",
      "token_auth_method": "client_secret_basic",
      "grant_types": [
        "string"
      ],
      "response_types": [
        "string"
      ],
      "redirect_uris": [
        "https://app.example.com/callback"
      ],
      "post_logout_redirect_uris": [
        "https://app.example.com/callback"
      ],
      "scopes": [
        "openid profile email"
      ],
      "require_pkce": true,
      "access_token_lifetime": 3600,
      "id_token_lifetime": 3600,
      "refresh_token_lifetime": 3600,
      "assert_roles": true,
      "require_role_assignment": true,
      "is_enabled": true,
      "allow_registration": true,
      "login_flow_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "role_count": 1,
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
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## GET /api/v1/admin/applications/:id {#get-api-v1-admin-applications-id}

Returns one application.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/applications/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/applications/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/applications/{id}",
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
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
  "application": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "name": "Example",
    "description": "string",
    "type": "m2m",
    "logo_url": "https://example.com/logo.png",
    "website_url": "https://example.com",
    "privacy_url": "https://example.com",
    "terms_url": "https://example.com",
    "client_id": "shop-web",
    "client_id_issued_at": 1790000000,
    "has_secret": true,
    "secret_hint": "string",
    "secret_created_at": "2026-09-28T09:30:00Z",
    "token_auth_method": "client_secret_basic",
    "grant_types": [
      "string"
    ],
    "response_types": [
      "string"
    ],
    "redirect_uris": [
      "https://app.example.com/callback"
    ],
    "post_logout_redirect_uris": [
      "https://app.example.com/callback"
    ],
    "scopes": [
      "openid profile email"
    ],
    "require_pkce": true,
    "access_token_lifetime": 3600,
    "id_token_lifetime": 3600,
    "refresh_token_lifetime": 3600,
    "assert_roles": true,
    "require_role_assignment": true,
    "is_enabled": true,
    "allow_registration": true,
    "login_flow_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "role_count": 1,
    "created_at": "2026-09-28T09:30:00Z",
    "updated_at": "2026-09-28T09:30:00Z"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## PATCH /api/v1/admin/applications/:id {#patch-api-v1-admin-applications-id}

Replaces an application's settings.

Its type and client id stay as they are: every copy of the app already in use depends on both.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/applications/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .header("Content-Type", "application/json")
        .method("PATCH", BodyPublishers.ofString("""
                {
                  "name": "Example"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"name": "Example"
}`)
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/applications/"+id, body)
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
    f"{ADMIN_API}/api/v1/admin/applications/{id}",
    json={
        "name": "Example",
    },
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
| `name` <small>required</small> | string | At most 100 characters. |
| `description` | string | At most 255 characters. |
| `type` | string |  |
| `logo_url` | string | At most 512 characters. |
| `website_url` | string | At most 512 characters. |
| `privacy_url` | string | At most 512 characters. |
| `terms_url` | string | At most 512 characters. |
| `token_auth_method` | string |  |
| `grant_types` | array of string |  |
| `redirect_uris` | array of string | At most 512 items. |
| `post_logout_redirect_uris` | array of string | At most 512 items. |
| `scopes` | array of string |  |
| `require_pkce` | boolean or null |  |
| `access_token_lifetime` | integer |  |
| `id_token_lifetime` | integer |  |
| `refresh_token_lifetime` | integer |  |
| `assert_roles` | boolean or null | The flags may be left out: a new application is then enabled, requires PKCE and puts roles in its tokens; an existing one keeps its settings. |
| `require_role_assignment` | boolean or null |  |
| `is_enabled` | boolean or null |  |
| `allow_registration` | boolean or null | `allow_registration` may be left out too: a new application then offers registration on its sign-in page. |
| `login_flow_id` | string or null | `login_flow_id`: "" returns to the default flow, omitted keeps the current one. |

#### Response

`200 OK`

```json
{
  "application": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "name": "Example",
    "description": "string",
    "type": "m2m",
    "logo_url": "https://example.com/logo.png",
    "website_url": "https://example.com",
    "privacy_url": "https://example.com",
    "terms_url": "https://example.com",
    "client_id": "shop-web",
    "client_id_issued_at": 1790000000,
    "has_secret": true,
    "secret_hint": "string",
    "secret_created_at": "2026-09-28T09:30:00Z",
    "token_auth_method": "client_secret_basic",
    "grant_types": [
      "string"
    ],
    "response_types": [
      "string"
    ],
    "redirect_uris": [
      "https://app.example.com/callback"
    ],
    "post_logout_redirect_uris": [
      "https://app.example.com/callback"
    ],
    "scopes": [
      "openid profile email"
    ],
    "require_pkce": true,
    "access_token_lifetime": 3600,
    "id_token_lifetime": 3600,
    "refresh_token_lifetime": 3600,
    "assert_roles": true,
    "require_role_assignment": true,
    "is_enabled": true,
    "allow_registration": true,
    "login_flow_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "role_count": 1,
    "created_at": "2026-09-28T09:30:00Z",
    "updated_at": "2026-09-28T09:30:00Z"
  },
  "client_secret": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`application_login_flow_unknown`](/reference/errors#application_login_flow_unknown) | There is no such login flow. |
| 400 | [`application_too_many_uris`](/reference/errors#application_too_many_uris) | An application may have at most {max} redirect URIs of each kind. |
| 400 | [`application_type_fixed`](/reference/errors#application_type_fixed) | An application's type cannot be changed: register a new application instead. |
| 400 | [`application_type_invalid`](/reference/errors#application_type_invalid) | The type must be one of: web, spa, native, m2m. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 403 | [`system_application_change`](/reference/errors#system_application_change) | Only a super admin can change an application that has access to the admin API, or rotate its secret. |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## POST /api/v1/admin/applications/:id/secret {#post-api-v1-admin-applications-id-secret}

Replaces a confidential client's secret.

The old one stops working at once, so the app has to be given the new one straight away.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/applications/$ID/secret" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications/" + id + "/secret"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .POST(BodyPublishers.noBody())
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/applications/"+id+"/secret", nil)
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
response = requests.post(
    f"{ADMIN_API}/api/v1/admin/applications/{id}/secret",
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
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
  "application": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "name": "Example",
    "description": "string",
    "type": "m2m",
    "logo_url": "https://example.com/logo.png",
    "website_url": "https://example.com",
    "privacy_url": "https://example.com",
    "terms_url": "https://example.com",
    "client_id": "shop-web",
    "client_id_issued_at": 1790000000,
    "has_secret": true,
    "secret_hint": "string",
    "secret_created_at": "2026-09-28T09:30:00Z",
    "token_auth_method": "client_secret_basic",
    "grant_types": [
      "string"
    ],
    "response_types": [
      "string"
    ],
    "redirect_uris": [
      "https://app.example.com/callback"
    ],
    "post_logout_redirect_uris": [
      "https://app.example.com/callback"
    ],
    "scopes": [
      "openid profile email"
    ],
    "require_pkce": true,
    "access_token_lifetime": 3600,
    "id_token_lifetime": 3600,
    "refresh_token_lifetime": 3600,
    "assert_roles": true,
    "require_role_assignment": true,
    "is_enabled": true,
    "allow_registration": true,
    "login_flow_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "role_count": 1,
    "created_at": "2026-09-28T09:30:00Z",
    "updated_at": "2026-09-28T09:30:00Z"
  },
  "client_secret": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`application_no_secret`](/reference/errors#application_no_secret) | A public client has no secret to rotate. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 403 | [`system_application_change`](/reference/errors#system_application_change) | Only a super admin can change an application that has access to the admin API, or rotate its secret. |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## GET /api/v1/admin/applications/:id/apis {#get-api-v1-admin-applications-id-apis}

Lists every API with what the application may do with it: whether it is authorised to ask for tokens for it, and which scopes.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/applications/$ID/apis" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications/" + id + "/apis"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/applications/"+id+"/apis", nil)
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
    f"{ADMIN_API}/api/v1/admin/applications/{id}/apis",
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
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
  "apis": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "name": "Example",
      "identifier": "string",
      "enforce_roles": true,
      "authorized": true,
      "scopes": [
        {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example",
          "description": "string",
          "allowed": true
        }
      ]
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
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## PUT /api/v1/admin/applications/:id/apis/:api {#put-api-v1-admin-applications-id-apis-api}

Lets the application request tokens for an API and replaces the scopes it may ask for: the ceiling on its tokens.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl -X PUT "$ADMIN_API/api/v1/admin/applications/$ID/apis/$API" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scopes":["0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d"]}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications/" + id + "/apis/" + api))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .header("Content-Type", "application/json")
        .PUT(BodyPublishers.ofString("""
                {
                  "scopes": [
                    "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d"
                  ]
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"scopes": [
		"0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d"
	]
}`)
req, err := http.NewRequest(http.MethodPut, adminAPI+"/api/v1/admin/applications/"+id+"/apis/"+api, body)
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
response = requests.put(
    f"{ADMIN_API}/api/v1/admin/applications/{id}/apis/{api}",
    json={
        "scopes": [
            "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
        ],
    },
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `id` <small>required</small> | path |  |
| `api` <small>required</small> | path |  |

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `scopes` | array of string (uuid) |  |

#### Response

`200 OK`

```json
{
  "apis": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "name": "Example",
      "identifier": "string",
      "enforce_roles": true,
      "authorized": true,
      "scopes": [
        {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example",
          "description": "string",
          "allowed": true
        }
      ]
    }
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`application_api_scope_foreign`](/reference/errors#application_api_scope_foreign) | One of the scopes is not a scope of {api}. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 403 | [`system_api_access`](/reference/errors#system_api_access) | Only a super admin can give an application access to the server's own APIs, or take it away. |
| 404 | [`api_not_found`](/reference/errors#api_not_found) | There is no such API. |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## DELETE /api/v1/admin/applications/:id/apis/:api {#delete-api-v1-admin-applications-id-apis-api}

Stops the application asking for tokens for an API.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/applications/$ID/apis/$API" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications/" + id + "/apis/" + api))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/applications/"+id+"/apis/"+api, nil)
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
    f"{ADMIN_API}/api/v1/admin/applications/{id}/apis/{api}",
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `id` <small>required</small> | path |  |
| `api` <small>required</small> | path |  |

#### Response

`200 OK`

```json
{
  "apis": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "name": "Example",
      "identifier": "string",
      "enforce_roles": true,
      "authorized": true,
      "scopes": [
        {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example",
          "description": "string",
          "allowed": true
        }
      ]
    }
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 403 | [`system_api_access`](/reference/errors#system_api_access) | Only a super admin can give an application access to the server's own APIs, or take it away. |
| 404 | [`api_not_found`](/reference/errors#api_not_found) | There is no such API. |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## POST /api/v1/admin/applications/:id/token-preview {#post-api-v1-admin-applications-id-token-preview}

Runs the token endpoint's evaluation without issuing anything: whether a token is issued, each scope decision, and the claims.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/applications/$ID/token-preview" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d","audience":"string","scope":"openid profile email"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications/" + id + "/token-preview"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "user_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
                  "audience": "string",
                  "scope": "openid profile email"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"user_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
	"audience": "string",
	"scope": "openid profile email"
}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/applications/"+id+"/token-preview", body)
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
    f"{ADMIN_API}/api/v1/admin/applications/{id}/token-preview",
    json={
        "user_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
        "audience": "string",
        "scope": "openid profile email",
    },
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
| `user_id` | string (uuid) or null |  |
| `audience` | string |  |
| `scope` | string |  |

#### Response

`200 OK`

```json
{
  "preview": {
    "issued": true,
    "reason": "string",
    "decisions": [
      {
        "scope": "openid profile email",
        "kind": "string",
        "granted": true,
        "reason": "string",
        "is_default": true
      }
    ],
    "access_token_header": {},
    "access_token": {},
    "id_token": {}
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`token_preview_audience_unknown`](/reference/errors#token_preview_audience_unknown) | No API has that identifier. |
| 400 | [`token_preview_user_unknown`](/reference/errors#token_preview_user_unknown) | There is no such user to preview a token for. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## POST /api/v1/admin/applications {#post-api-v1-admin-applications}

Registers an application.

A confidential client is given its secret here, in the answer, and never again: only its hash is kept.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.write`](/reference/permissions#applications-write) |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/applications" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "name": "Example"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"name": "Example"
}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/applications", body)
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
    f"{ADMIN_API}/api/v1/admin/applications",
    json={
        "name": "Example",
    },
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `name` <small>required</small> | string | At most 100 characters. |
| `description` | string | At most 255 characters. |
| `type` | string |  |
| `logo_url` | string | At most 512 characters. |
| `website_url` | string | At most 512 characters. |
| `privacy_url` | string | At most 512 characters. |
| `terms_url` | string | At most 512 characters. |
| `token_auth_method` | string |  |
| `grant_types` | array of string |  |
| `redirect_uris` | array of string | At most 512 items. |
| `post_logout_redirect_uris` | array of string | At most 512 items. |
| `scopes` | array of string |  |
| `require_pkce` | boolean or null |  |
| `access_token_lifetime` | integer |  |
| `id_token_lifetime` | integer |  |
| `refresh_token_lifetime` | integer |  |
| `assert_roles` | boolean or null | The flags may be left out: a new application is then enabled, requires PKCE and puts roles in its tokens; an existing one keeps its settings. |
| `require_role_assignment` | boolean or null |  |
| `is_enabled` | boolean or null |  |
| `allow_registration` | boolean or null | `allow_registration` may be left out too: a new application then offers registration on its sign-in page. |
| `login_flow_id` | string or null | `login_flow_id`: "" returns to the default flow, omitted keeps the current one. |

#### Response

`201 Created`

```json
{
  "application": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "name": "Example",
    "description": "string",
    "type": "m2m",
    "logo_url": "https://example.com/logo.png",
    "website_url": "https://example.com",
    "privacy_url": "https://example.com",
    "terms_url": "https://example.com",
    "client_id": "shop-web",
    "client_id_issued_at": 1790000000,
    "has_secret": true,
    "secret_hint": "string",
    "secret_created_at": "2026-09-28T09:30:00Z",
    "token_auth_method": "client_secret_basic",
    "grant_types": [
      "string"
    ],
    "response_types": [
      "string"
    ],
    "redirect_uris": [
      "https://app.example.com/callback"
    ],
    "post_logout_redirect_uris": [
      "https://app.example.com/callback"
    ],
    "scopes": [
      "openid profile email"
    ],
    "require_pkce": true,
    "access_token_lifetime": 3600,
    "id_token_lifetime": 3600,
    "refresh_token_lifetime": 3600,
    "assert_roles": true,
    "require_role_assignment": true,
    "is_enabled": true,
    "allow_registration": true,
    "login_flow_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "role_count": 1,
    "created_at": "2026-09-28T09:30:00Z",
    "updated_at": "2026-09-28T09:30:00Z"
  },
  "client_secret": "string"
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`application_login_flow_unknown`](/reference/errors#application_login_flow_unknown) | There is no such login flow. |
| 400 | [`application_too_many_uris`](/reference/errors#application_too_many_uris) | An application may have at most {max} redirect URIs of each kind. |
| 400 | [`application_type_invalid`](/reference/errors#application_type_invalid) | The type must be one of: web, spa, native, m2m. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## DELETE /api/v1/admin/applications/:id {#delete-api-v1-admin-applications-id}

Removes an application, the roles it defines, and everyone's hold on them.

Tokens it has already issued stay valid until they expire.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`applications.write`](/reference/permissions#applications-write) |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/applications/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/applications/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/applications/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/applications/{id}",
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
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |
| 409 | [`system_application`](/reference/errors#system_application) | admin-cli is part of the server and cannot be deleted. Turn it off instead. |
