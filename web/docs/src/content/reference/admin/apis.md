---
title: "APIs"
description: "The APIs access tokens are issued for, and their scopes."
order: 17
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/apis {#get-api-v1-admin-apis}

Returns every API matching the search, sorted by name, with its scopes and how many applications may use it.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`apis.read`](/reference/permissions#apis-read) or [`applications.write`](/reference/permissions#applications-write) for at least one application |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/apis?search=value" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/apis" + "?search=value"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/apis"+"?search=value", nil)
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
    f"{ADMIN_API}/api/v1/admin/apis",
    params={
        "search": "value",
    },
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `search` | query |  |

#### Response

`200 OK`

```json
{
  "apis": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "name": "Example",
      "identifier": "string",
      "description": "string",
      "enforce_roles": true,
      "scopes": [
        {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example",
          "description": "string",
          "is_default": true
        }
      ],
      "system": "string",
      "signing_algorithm": "string",
      "token_lifetime": 3600,
      "allow_offline_access": true,
      "application_count": 1,
      "role_count": 1,
      "issuer": "https://example.com",
      "jwks_uri": "https://example.com",
      "created_at": "2026-09-28T09:30:00Z",
      "updated_at": "2026-09-28T09:30:00Z"
    }
  ],
  "total": 1
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## GET /api/v1/admin/apis/:id {#get-api-v1-admin-apis-id}

Returns one API.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`apis.read`](/reference/permissions#apis-read) or [`applications.write`](/reference/permissions#applications-write) for at least one application |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/apis/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/apis/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/apis/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/apis/{id}",
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
  "api": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "name": "Example",
    "identifier": "string",
    "description": "string",
    "enforce_roles": true,
    "scopes": [
      {
        "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
        "name": "Example",
        "description": "string",
        "is_default": true
      }
    ],
    "system": "string",
    "signing_algorithm": "string",
    "token_lifetime": 3600,
    "allow_offline_access": true,
    "application_count": 1,
    "role_count": 1,
    "issuer": "https://example.com",
    "jwks_uri": "https://example.com",
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
| 404 | [`api_not_found`](/reference/errors#api_not_found) | There is no such API. |

## GET /api/v1/admin/apis/:id/applications {#get-api-v1-admin-apis-id-applications}

Lists visible applications with their access to the API.

Access is changed through the application's endpoints.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`apis.read`](/reference/permissions#apis-read) or [`applications.write`](/reference/permissions#applications-write) for at least one application |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/apis/$ID/applications" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/apis/" + id + "/applications"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/apis/"+id+"/applications", nil)
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
    f"{ADMIN_API}/api/v1/admin/apis/{id}/applications",
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

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`api_not_found`](/reference/errors#api_not_found) | There is no such API. |

## GET /api/v1/admin/apis/:id/logs {#get-api-v1-admin-apis-id-logs}

Lists changes to the API and applications gaining or losing access, newest first.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`apis.read`](/reference/permissions#apis-read) |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/apis/$ID/logs" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/apis/" + id + "/logs"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/apis/"+id+"/logs", nil)
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
    f"{ADMIN_API}/api/v1/admin/apis/{id}/logs",
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

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`api_not_found`](/reference/errors#api_not_found) | There is no such API. |

## POST /api/v1/admin/apis {#post-api-v1-admin-apis}

Registers an API with its scopes.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`apis.write`](/reference/permissions#apis-write) |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/apis" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/apis"))
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
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/apis", body)
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
    f"{ADMIN_API}/api/v1/admin/apis",
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
| `identifier` | string | At most 255 characters. |
| `description` | string | At most 255 characters. |
| `enforce_roles` | boolean or null | `enforce_roles` and `allow_offline_access` may be left out: a new API then enforces roles and refuses refresh tokens; an existing one keeps them. |
| `signing_algorithm` | string | `signing_algorithm` left empty is RS256. `token_lifetime` is in seconds, and zero leaves it to each application. |
| `token_lifetime` | integer |  |
| `allow_offline_access` | boolean or null |  |
| `scopes` | array of ScopeRequest |  |
| `scopes[].id` | string (uuid) |  |
| `scopes[].name` | string |  |
| `scopes[].description` | string |  |
| `scopes[].is_default` | boolean |  |

#### Response

Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{"error", "error_description"}`.

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`api_scope_foreign`](/reference/errors#api_scope_foreign) | {scope} is not a scope of this API. |
| 400 | [`api_too_many_scopes`](/reference/errors#api_too_many_scopes) | An API may have at most {max} scopes. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 409 | [`api_identifier_taken`](/reference/errors#api_identifier_taken) | An API with this identifier already exists. |

## PATCH /api/v1/admin/apis/:id {#patch-api-v1-admin-apis-id}

Replaces an API's name, description, role enforcement and scopes.

The identifier never changes.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`apis.write`](/reference/permissions#apis-write) |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/apis/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/apis/" + id))
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
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/apis/"+id, body)
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
    f"{ADMIN_API}/api/v1/admin/apis/{id}",
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
| `identifier` | string | At most 255 characters. |
| `description` | string | At most 255 characters. |
| `enforce_roles` | boolean or null | `enforce_roles` and `allow_offline_access` may be left out: a new API then enforces roles and refuses refresh tokens; an existing one keeps them. |
| `signing_algorithm` | string | `signing_algorithm` left empty is RS256. `token_lifetime` is in seconds, and zero leaves it to each application. |
| `token_lifetime` | integer |  |
| `allow_offline_access` | boolean or null |  |
| `scopes` | array of ScopeRequest |  |
| `scopes[].id` | string (uuid) |  |
| `scopes[].name` | string |  |
| `scopes[].description` | string |  |
| `scopes[].is_default` | boolean |  |

#### Response

Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{"error", "error_description"}`.

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`api_identifier_fixed`](/reference/errors#api_identifier_fixed) | An API's identifier cannot be changed: register a new API instead. |
| 400 | [`api_scope_foreign`](/reference/errors#api_scope_foreign) | {scope} is not a scope of this API. |
| 400 | [`api_too_many_scopes`](/reference/errors#api_too_many_scopes) | An API may have at most {max} scopes. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`api_not_found`](/reference/errors#api_not_found) | There is no such API. |
| 409 | [`api_identifier_taken`](/reference/errors#api_identifier_taken) | An API with this identifier already exists. |

## DELETE /api/v1/admin/apis/:id {#delete-api-v1-admin-apis-id}

Removes an API with its scopes, application authorisations and role grants.

Issued tokens stay valid until they expire.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`apis.write`](/reference/permissions#apis-write) |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/apis/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/apis/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/apis/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/apis/{id}",
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
| 404 | [`api_not_found`](/reference/errors#api_not_found) | There is no such API. |
| 409 | [`system_api`](/reference/errors#system_api) | This API is part of the server and cannot be deleted. |
