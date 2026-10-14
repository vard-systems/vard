---
title: "Administrator roles"
description: "What administrators may do in the panel."
order: 23
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/admin-permissions {#get-api-v1-admin-admin-permissions}

Returns the catalog roles pick from, in the order the panel lists it.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/admin-permissions" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admin-permissions"))
        .header("Cookie", "loginer_session=" + SESSION)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/admin-permissions", nil)
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
    f"{ADMIN_API}/api/v1/admin/admin-permissions",
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "permissions": [
    {
      "name": "Example",
      "group": "string",
      "description": "string",
      "scopable": true
    }
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## GET /api/v1/admin/admin-roles {#get-api-v1-admin-admin-roles}

Returns every admin role matching the search, sorted by name, with how many administrators hold each.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/admin-roles?search=value" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admin-roles" + "?search=value"))
        .header("Cookie", "loginer_session=" + SESSION)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/admin-roles"+"?search=value", nil)
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
    f"{ADMIN_API}/api/v1/admin/admin-roles",
    params={
        "search": "value",
    },
    cookies={"loginer_session": SESSION},
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
  "roles": [
    {
      "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
      "name": "Example",
      "description": "string",
      "permissions": [
        "string"
      ],
      "is_builtin": true,
      "admin_count": 1,
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
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## POST /api/v1/admin/admin-roles {#post-api-v1-admin-admin-roles}

Adds an admin role.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/admin-roles" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example","description":"string","permissions":["string"]}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admin-roles"))
        .header("Cookie", "loginer_session=" + SESSION)
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "name": "Example",
                  "description": "string",
                  "permissions": [
                    "string"
                  ]
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"name": "Example",
	"description": "string",
	"permissions": [
		"string"
	]
}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/admin-roles", body)
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
    f"{ADMIN_API}/api/v1/admin/admin-roles",
    json={
        "name": "Example",
        "description": "string",
        "permissions": [
            "string",
        ],
    },
    cookies={"loginer_session": SESSION},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `name` <small>required</small> | string | Adminrole. |
| `description` | string | At most 255 characters. |
| `permissions` | array of string |  |

#### Response

`201 Created`

```json
{
  "role": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "name": "Example",
    "description": "string",
    "permissions": [
      "string"
    ],
    "is_builtin": true,
    "admin_count": 1,
    "created_at": "2026-09-28T09:30:00Z",
    "updated_at": "2026-09-28T09:30:00Z"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 409 | [`admin_role_name_taken`](/reference/errors#admin_role_name_taken) | An admin role with this name already exists. |

## PATCH /api/v1/admin/admin-roles/:id {#patch-api-v1-admin-admin-roles-id}

Replaces an admin role's name, description and permissions.

Every administrator holding it has the new permissions on their next request.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/admin-roles/$ID" \
  -b "loginer_session=$SESSION" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example","description":"string","permissions":["string"]}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admin-roles/" + id))
        .header("Cookie", "loginer_session=" + SESSION)
        .header("Content-Type", "application/json")
        .method("PATCH", BodyPublishers.ofString("""
                {
                  "name": "Example",
                  "description": "string",
                  "permissions": [
                    "string"
                  ]
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"name": "Example",
	"description": "string",
	"permissions": [
		"string"
	]
}`)
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/admin-roles/"+id, body)
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
    f"{ADMIN_API}/api/v1/admin/admin-roles/{id}",
    json={
        "name": "Example",
        "description": "string",
        "permissions": [
            "string",
        ],
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
| `name` <small>required</small> | string | Adminrole. |
| `description` | string | At most 255 characters. |
| `permissions` | array of string |  |

#### Response

`200 OK`

```json
{
  "role": {
    "id": "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d",
    "name": "Example",
    "description": "string",
    "permissions": [
      "string"
    ],
    "is_builtin": true,
    "admin_count": 1,
    "created_at": "2026-09-28T09:30:00Z",
    "updated_at": "2026-09-28T09:30:00Z"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 403 | [`admin_role_built_in`](/reference/errors#admin_role_built_in) | super_admin is built in and cannot be changed or removed. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`admin_role_not_found`](/reference/errors#admin_role_not_found) | There is no such admin role. |
| 409 | [`admin_role_name_taken`](/reference/errors#admin_role_name_taken) | An admin role with this name already exists. |

## DELETE /api/v1/admin/admin-roles/:id {#delete-api-v1-admin-admin-roles-id}

Removes an admin role.

The administrators who held it keep their other roles.

| | |
| --- | --- |
| Auth | Admin session · super admins only |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/admin-roles/$ID" \
  -b "loginer_session=$SESSION"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/admin-roles/" + id))
        .header("Cookie", "loginer_session=" + SESSION)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/admin-roles/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/admin-roles/{id}",
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
| 403 | [`admin_role_built_in`](/reference/errors#admin_role_built_in) | super_admin is built in and cannot be changed or removed. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`admin_role_not_found`](/reference/errors#admin_role_not_found) | There is no such admin role. |
