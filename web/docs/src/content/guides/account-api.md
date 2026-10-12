---
title: User self-service
description: Call the Account API — the one behind the sign-in app's account pages — for a signed-in user, from your own application.
order: 2
section: Manage users
nav: 'User self-service'
icon: user-settings
---

| Guide                                        | Who acts                           | Manages         | Use for                                                |
| -------------------------------------------- | ---------------------------------- | --------------- | ------------------------------------------------------ |
| **[With admin-cli](/guides/admin-api)**      | Your backend, as a service         | Any user        | Provisioning, admin tools, syncing from another system |
| **[User self-service](/guides/account-api)** | A signed-in user, through your app | Only themselves | Profile and security screens inside your app           |

The sign-in app's account pages let a user edit their profile, change their password and email, see where they're signed in and which apps they've connected. All of that is the **Account API**, and your application can call it too — for a user who signed in to it — to build those screens into your own app.

## How it works

1. Your application signs the user in as usual, asking for an access token for the Account API (audience `{{accountAPI}}`).
2. It calls `$ISSUER/api/v1/account/...` with that token.
3. The token acts for that user only, within its scopes:

| Scope           | Allows                                                                        |
| --------------- | ----------------------------------------------------------------------------- |
| `account.read`  | Reading the profile, sessions and connected applications                      |
| `account.write` | Changing the profile, password and email; ending sessions; disconnecting apps |

## 1. Allow your application

In the panel, open **Applications → your application → API access**, and under **Account API** tick `account.read` and, if you'll change anything, `account.write`.

## 2. Sign the user in for the Account API

Add `audience` to the [authorization request](/guides/authorization-code), and `account.write` to `scope` if you need it (`account.read` comes by default):

```http
GET $ISSUER/oauth2/authorize?response_type=code
  &client_id=CLIENT_ID
  &redirect_uri=https://app.example.com/callback
  &scope=openid profile account.write
  &audience={{accountAPI}}
  &state=STATE&code_challenge=CHALLENGE&code_challenge_method=S256
```

Exchange the code as usual. The access token you get is the Account API token.

## 3. Call it

### Read the user

```bash title="cURL"
curl "$ISSUER/api/v1/account/me" -H "Authorization: Bearer $ACCESS_TOKEN"
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder(URI.create(ISSUER + "/api/v1/account/me"))
        .header("Authorization", "Bearer " + accessToken)
        .build();
String me = HttpClient.newHttpClient().send(request, BodyHandlers.ofString()).body();
```

```go title="Go"
req, _ := http.NewRequest(http.MethodGet, issuer+"/api/v1/account/me", nil)
req.Header.Set("Authorization", "Bearer "+accessToken)
res, err := http.DefaultClient.Do(req)
```

```python title="Python"
me = requests.get(
    f"{ISSUER}/api/v1/account/me",
    headers={"Authorization": f"Bearer {access_token}"},
).json()["user"]
```

### Change their name

```bash title="cURL"
curl -X PATCH "$ISSUER/api/v1/account/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"first_name":"Ada","last_name":"Lovelace"}'
```

```java title="Java"
HttpRequest request = HttpRequest.newBuilder(URI.create(ISSUER + "/api/v1/account/me"))
        .header("Authorization", "Bearer " + accessToken)
        .header("Content-Type", "application/json")
        .method("PATCH", BodyPublishers.ofString("""
                {"first_name": "Ada", "last_name": "Lovelace"}
                """))
        .build();
HttpResponse<String> response = HttpClient.newHttpClient().send(request, BodyHandlers.ofString());
```

```go title="Go"
body := strings.NewReader(`{"first_name": "Ada", "last_name": "Lovelace"}`)
req, _ := http.NewRequest(http.MethodPatch, issuer+"/api/v1/account/me", body)
req.Header.Set("Authorization", "Bearer "+accessToken)
req.Header.Set("Content-Type", "application/json")
res, err := http.DefaultClient.Do(req)
```

```python title="Python"
requests.patch(
    f"{ISSUER}/api/v1/account/me",
    headers={"Authorization": f"Bearer {access_token}"},
    json={"first_name": "Ada", "last_name": "Lovelace"},
).raise_for_status()
```

### Everything it can do

| Do                           | Call                                                                    | Scope           |
| ---------------------------- | ----------------------------------------------------------------------- | --------------- |
| Read the profile             | `GET /api/v1/account/me`                                                | `account.read`  |
| Change the name              | `PATCH /api/v1/account/me`                                              | `account.write` |
| Change the password          | `POST /api/v1/account/password` with `current_password`, `new_password` | `account.write` |
| List where they're signed in | `GET /api/v1/account/sessions`                                          | `account.read`  |
| Sign one browser out         | `DELETE /api/v1/account/sessions/{id}`                                  | `account.write` |
| List connected applications  | `GET /api/v1/account/connected-applications`                            | `account.read`  |
| Disconnect an application    | `DELETE /api/v1/account/connected-applications/{client_id}`             | `account.write` |

Each is in the [Account reference](/reference/public/account), with its fields and errors.

## Good to know

- **The same API, two ways in.** The sign-in app calls these routes with the user's session cookie; your application calls them with a token. A request with a token is judged by the token alone.
- **Changing the password signs the user out everywhere** and revokes every application's refresh tokens for them — yours included. Expect to sign them in again.
- **The email cannot be changed with a token.** A new address plus a password reset is the whole account, so `POST /api/v1/account/email` takes the user's session in the sign-in app and their current password; a token gets `403 email_change_not_offered`.
- **No session is "current"** for a token: the sessions list marks none, and any of them can be ended.
- **Errors:** `401 account_token_refused` for a token that expired, was issued for another API, or whose user can no longer sign in; `403 account_scope_missing` when it lacks the scope a route needs.
- **From a browser app** on another origin, add that origin to `{{envPrefix}}CORS_ORIGINS` so the browser may send the token.
