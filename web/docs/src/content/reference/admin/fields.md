---
title: "User fields"
description: "The custom fields a user record is made of."
order: 8
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/user-fields {#get-api-v1-admin-user-fields}

Returns the fields a user record has, in the order they are shown.

The panel builds both its table and its form from this.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`users.read`](/reference/permissions#users-read) |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/user-fields" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-fields"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/user-fields", nil)
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
    f"{ADMIN_API}/api/v1/admin/user-fields",
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "fields": [
    {
      "id": "string",
      "name": "Example",
      "label": "string",
      "type": "bool",
      "is_required": true,
      "is_unique": true,
      "min": 1,
      "max": 1,
      "starts_with": "string",
      "position": 1,
      "is_builtin": true
    }
  ],
  "types": [
    "bool"
  ]
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |

## POST /api/v1/admin/user-fields {#post-api-v1-admin-user-fields}

Adds a field to every user record.

Existing users simply have no value for it until they are edited.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`user_fields.write`](/reference/permissions#user_fields-write) |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/user-fields" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Example","type":"string"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-fields"))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "name": "Example",
                  "type": "string"
                }
                """))
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{
	"name": "Example",
	"type": "string"
}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/user-fields", body)
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
    f"{ADMIN_API}/api/v1/admin/user-fields",
    json={
        "name": "Example",
        "type": "string",
    },
    headers={"Authorization": f"Bearer {ADMIN_TOKEN}"},
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `label` | string | At most 100 characters. |
| `is_required` | boolean |  |
| `is_unique` | boolean |  |
| `min` | number or null |  |
| `max` | number or null |  |
| `starts_with` | string | At most 64 characters. |
| `name` <small>required</small> | string | Column. |
| `type` <small>required</small> | string | Fieldtype. |

#### Response

`201 Created`

```json
{
  "field": {
    "id": "string",
    "name": "Example",
    "label": "string",
    "type": "bool",
    "is_required": true,
    "is_unique": true,
    "min": 1,
    "max": 1,
    "starts_with": "string",
    "position": 1,
    "is_builtin": true
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 409 | [`user_field_name_taken`](/reference/errors#user_field_name_taken) | A field with this name already exists. |

## PATCH /api/v1/admin/user-fields/:id {#patch-api-v1-admin-user-fields-id}

Changes a field's rules.

Name and type never change, since records hold values under them.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`user_fields.write`](/reference/permissions#user_fields-write) |

```bash title="cURL"
curl -X PATCH "$ADMIN_API/api/v1/admin/user-fields/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-fields/" + id))
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
req, err := http.NewRequest(http.MethodPatch, adminAPI+"/api/v1/admin/user-fields/"+id, body)
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
    f"{ADMIN_API}/api/v1/admin/user-fields/{id}",
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
| `label` | string | At most 100 characters. |
| `is_required` | boolean |  |
| `is_unique` | boolean |  |
| `min` | number or null |  |
| `max` | number or null |  |
| `starts_with` | string | At most 64 characters. |

#### Response

`200 OK`

```json
{
  "field": {
    "id": "string",
    "name": "Example",
    "label": "string",
    "type": "bool",
    "is_required": true,
    "is_unique": true,
    "min": 1,
    "max": 1,
    "starts_with": "string",
    "position": 1,
    "is_builtin": true
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 401 | [`not_signed_in`](/reference/errors#not_signed_in) | You are not signed in. |
| 401 | [`token_refused`](/reference/errors#token_refused) | This access token is not accepted by the admin API: it has expired, is for another API, is not a service's token, or its application has lost access. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 403 | [`forbidden`](/reference/errors#forbidden) | You do not have permission to do that. |
| 404 | [`user_field_not_found`](/reference/errors#user_field_not_found) | There is no such field. |

## DELETE /api/v1/admin/user-fields/:id {#delete-api-v1-admin-user-fields-id}

Removes a field; stored values are dropped when each user is next saved.

| | |
| --- | --- |
| Auth | Admin session or [admin-cli token](/guides/admin-api) · needs [`user_fields.write`](/reference/permissions#user_fields-write) |

```bash title="cURL"
curl -X DELETE "$ADMIN_API/api/v1/admin/user-fields/$ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/user-fields/" + id))
        .header("Authorization", "Bearer " + ADMIN_TOKEN)
        .DELETE()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodDelete, adminAPI+"/api/v1/admin/user-fields/"+id, nil)
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
    f"{ADMIN_API}/api/v1/admin/user-fields/{id}",
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
| 404 | [`user_field_not_found`](/reference/errors#user_field_not_found) | There is no such field. |
