---
title: "First administrator"
description: "Creating the first administrator of a new installation."
order: 2
generated: true
---

<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->

## GET /api/v1/admin/setup {#get-api-v1-admin-setup}

Says whether the panel still needs its first administrator.

| | |
| --- | --- |
| Auth | None |

```bash title="cURL"
curl "$ADMIN_API/api/v1/admin/setup"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/setup"))
        .GET()
        .build();

HttpResponse<String> response = HttpClient.newHttpClient()
        .send(request, BodyHandlers.ofString());
```

```go title="Go"
req, err := http.NewRequest(http.MethodGet, adminAPI+"/api/v1/admin/setup", nil)
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
response = requests.get(f"{ADMIN_API}/api/v1/admin/setup")
response.raise_for_status()
```

#### Response

`200 OK`

```json
{
  "required": true
}
```

## POST /api/v1/admin/setup {#post-api-v1-admin-setup}

Makes the first administrator: a super admin, with the address and password whoever is setting the panel up chose.

| | |
| --- | --- |
| Auth | None |
| Rate limit | Per IP address |

```bash title="cURL"
curl -X POST "$ADMIN_API/api/v1/admin/setup" \
  -H "Content-Type: application/json" \
  -d '{"email":"ada@example.com","password":"correct horse battery staple","first_name":"Ada","last_name":"Lovelace"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder()
        .uri(URI.create(ADMIN_API + "/api/v1/admin/setup"))
        .header("Content-Type", "application/json")
        .POST(BodyPublishers.ofString("""
                {
                  "email": "ada@example.com",
                  "password": "correct horse battery staple",
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
	"email": "ada@example.com",
	"password": "correct horse battery staple",
	"first_name": "Ada",
	"last_name": "Lovelace"
}`)
req, err := http.NewRequest(http.MethodPost, adminAPI+"/api/v1/admin/setup", body)
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
    f"{ADMIN_API}/api/v1/admin/setup",
    json={
        "email": "ada@example.com",
        "password": "correct horse battery staple",
        "first_name": "Ada",
        "last_name": "Lovelace",
    },
)
response.raise_for_status()
```

#### Body

| Field | Type | Description |
| --- | --- | --- |
| `email` <small>required</small> | string | An email address, at most 255 characters. |
| `password` <small>required</small> | string | At least 10 characters, at most 128 characters. |
| `first_name` <small>required</small> | string | At most 100 characters. |
| `last_name` | string | At most 100 characters. |

#### Response

`201 Created`

```json
{
  "admin": {
    "id": "string",
    "email": "ada@example.com",
    "full_name": "string"
  }
}
```

#### Errors

| Status | Code | Means |
| --- | --- | --- |
| 400 | [`admin_password_too_long`](/reference/errors#admin_password_too_long) | The password is too long. |
| 400 | [`invalid_body`](/reference/errors#invalid_body) | The request could not be read. Reload the page and try again. |
| 403 | [`cross_origin`](/reference/errors#cross_origin) | Requests from {origin} are not allowed. |
| 409 | [`setup_done`](/reference/errors#setup_done) | This panel has already been set up. |
| 429 | [`rate_limited`](/reference/errors#rate_limited) | Too many attempts. Try again in {seconds} seconds. |
