---
title: "User roles"
description: "The roles users hold, globally and in each application."
order: 16
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/user-roles {#get-api-v1-admin-user-roles}

Returns a page of roles sorted by name, each with its member count and every role it includes. `scope` is "global", "application" or empty for both; `application` narrows to one application.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`users.read`](/reference/permissions#users-read) or [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/user-roles?search=value&scope=openid+profile+email&application=value" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-roles" + "?search=value&scope=openid+profile+email&application=value"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/user-roles"+"?search=value&scope=openid+profile+email&application=value", nil)
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
    f"{ADMIN_API}/api/v1/admin/user-roles",
    params={
        "search": "value",
        "scope": "openid profile email",
        "application": "value",
    },
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Parameters

| Name | In | Description |
| --- | --- | --- |
| `search` | query |  |
| `scope` | query | Space-separated scopes: `openid`, `profile`, `email`, `offline_access`, `roles`, and the scopes of the API named in `audience`. |
| `application` | query |  |

#### Response

`200 OK`

```json
{
  "roles": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "application_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "name": "Example",
      "description": "string",
      "is_default": true,
      "inherits": [
        {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example",
          "application_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d"
        }
      ],
      "inherited_roles": [
        {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example",
          "application_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d"
        }
      ],
      "api_scopes": [
        {
          "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
          "name": "Example",
          "api_id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d"
        }
      ],
      "user_count": 1,
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
| 400 | [`user_role_scope_invalid`](/reference/errors#user_role_scope_invalid) | The scope must be "global" or "application". |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`application_not_found`](/reference/errors#application_not_found) | There is no such application. |

## GET /api/v1/admin/user-roles/:id {#get-api-v1-admin-user-roles-id}

Returns one role.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`users.read`](/reference/permissions#users-read) or [`applications.read`](/reference/permissions#applications-read) for at least one application |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/user-roles/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-roles/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/user-roles/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/user-roles/{id}",
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
| 404 | [`user_role_not_found`](/reference/errors#user_role_not_found) | There is no such role. |

## POST /api/v1/admin/user-roles {#post-api-v1-admin-user-roles}

Adds a role: a global one when application_id is null, otherwise one of that application's.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`user_roles.write`](/reference/permissions#user_roles-write) for at least one application |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/user-roles" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-roles"))
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
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/user-roles", body)
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
    f"{ADMIN_API}/api/v1/admin/user-roles",
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
| `application_id` | string (uuid) or null | `application_id` is the application the role belongs to, or null for a global role. It is read on create only: a role keeps its scope. |
| `name` <small>required</small> | string | Rolename. |
| `description` | string | At most 255 characters. |
| `is_default` | boolean | `is_default` gives the role to every user created without roles of their own. |
| `inherits` | array of string (uuid) |  |
| `api_scopes` | array of string (uuid) or null | `api_scopes` replace the role's scope grants; left out, they are kept, so administrators who cannot see APIs can still edit roles. |

#### Response

Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{"error", "error_description"}`.

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 400 | [`user_role_application_unknown`](/reference/errors#user_role_application_unknown) | The role's application does not exist. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 409 | [`user_role_name_taken`](/reference/errors#user_role_name_taken) | A role with this name already exists in this scope. |

## PATCH /api/v1/admin/user-roles/:id {#patch-api-v1-admin-user-roles-id}

Replaces a role's name, description, default flag and inheritance.

Its scope never changes.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`user_roles.write`](/reference/permissions#user_roles-write) for at least one application |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/user-roles/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-roles/" + id))
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
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/user-roles/"+id, body)
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
    f"{ADMIN_API}/api/v1/admin/user-roles/{id}",
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
| `application_id` | string (uuid) or null | `application_id` is the application the role belongs to, or null for a global role. It is read on create only: a role keeps its scope. |
| `name` <small>required</small> | string | Rolename. |
| `description` | string | At most 255 characters. |
| `is_default` | boolean | `is_default` gives the role to every user created without roles of their own. |
| `inherits` | array of string (uuid) |  |
| `api_scopes` | array of string (uuid) or null | `api_scopes` replace the role's scope grants; left out, they are kept, so administrators who cannot see APIs can still edit roles. |

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
| 404 | [`user_role_not_found`](/reference/errors#user_role_not_found) | There is no such role. |
| 409 | [`user_role_name_taken`](/reference/errors#user_role_name_taken) | A role with this name already exists in this scope. |

## DELETE /api/v1/admin/user-roles/:id {#delete-api-v1-admin-user-roles-id}

Removes a role for good.

The users who held it keep everything else they hold, and the roles that included it simply stop.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`user_roles.write`](/reference/permissions#user_roles-write) for at least one application |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/user-roles/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-roles/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/user-roles/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/user-roles/{id}",
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
| 404 | [`user_role_not_found`](/reference/errors#user_role_not_found) | There is no such role. |
