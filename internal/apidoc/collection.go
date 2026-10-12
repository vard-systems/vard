package apidoc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"loginer/internal/brand"
)

const postmanSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

func (b *built) collection(server Server) ([]byte, error) {
	variables := []map[string]any{
		variable("issuer", "http://localhost:5173", "The public server's address: the issuer."),
		variable("client_id", "", "Your application's client ID."),
		variable("client_secret", "", "Your application's client secret."),
		variable("access_token", "", "A user's access token: for userinfo, or for the account API."),
	}
	if server.Key == "admin" {
		variables = []map[string]any{
			variable("issuer", "http://localhost:5173", "The public server's address, where tokens come from."),
			variable("admin_api", "http://localhost:5174", "The panel's address, where the admin API is reached."),
			variable("admin_cli_secret", "", "admin-cli's secret: rotate it in the panel and paste it here."),
			variable("admin_token", "", "Filled in by \"Get an admin-cli token\"."),
			variable("admin_session", "", "An administrator's session cookie, for the routes only a person may call."),
		}
	}

	var folders []any
	if server.Key == "admin" {
		folders = append(folders, map[string]any{
			"name":        "Start here",
			"description": "Get admin-cli's token first: it is stored in {{admin_token}}, which the other requests send.",
			"item":        []any{adminTokenRequest()},
		})
	}

	for _, section := range server.Sections {
		items := make([]any, 0, len(section.Operations))
		for _, op := range section.Operations {
			items = append(items, collectionRequest(server, op))
		}
		folders = append(folders, map[string]any{
			"name":        section.Title,
			"description": section.Intro,
			"item":        items,
		})
	}

	document := map[string]any{
		"info": map[string]any{
			"name":        brand.Name + " " + server.Title,
			"description": serverDescription(server.Key),
			"schema":      postmanSchema,
		},
		"variable": variables,
		"item":     folders,
	}

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func variable(key, value, description string) map[string]any {
	return map[string]any{"key": key, "value": value, "type": "string", "description": description}
}

func adminTokenRequest() map[string]any {
	return map[string]any{
		"name": "Get an admin-cli token",
		"event": []any{map[string]any{
			"listen": "test",
			"script": map[string]any{
				"type": "text/javascript",
				"exec": []string{`pm.collectionVariables.set("admin_token", pm.response.json().access_token);`},
			},
		}},
		"request": map[string]any{
			"method":      http.MethodPost,
			"description": "Client credentials for admin-cli, for the admin API. The token's scopes are the permissions admin-cli is allowed in the panel.",
			"auth": map[string]any{"type": "basic", "basic": []any{
				map[string]any{"key": "username", "value": "admin-cli", "type": "string"},
				map[string]any{"key": "password", "value": "{{admin_cli_secret}}", "type": "string"},
			}},
			"body": map[string]any{"mode": "urlencoded", "urlencoded": []any{
				map[string]any{"key": "grant_type", "value": "client_credentials", "type": "text"},
				map[string]any{"key": "audience", "value": brand.AdminAPIIdentifier, "type": "text"},
			}},
			"url": collectionURL("{{issuer}}", "/oauth2/token", nil),
		},
	}
}

func collectionRequest(server Server, op Operation) map[string]any {
	host := "{{issuer}}"
	if server.Key == "admin" {
		host = "{{admin_api}}"
	}

	request := map[string]any{
		"method": op.Method,
		"url":    collectionURL(host, op.Path, &op),
		"auth":   collectionAuth(op),
	}

	description := strings.TrimSpace(op.Summary + "\n\n" + op.Description)
	if description != "" {
		request["description"] = description
	}

	var headers []any
	if cookie := sessionCookie(op); cookie != "" {
		headers = append(headers, map[string]any{"key": "Cookie", "value": cookie, "type": "text"})
	}

	switch {
	case op.Body != nil:
		body, _ := json.MarshalIndent(example(op.Body, "", 0, true), "", "  ")
		headers = append(headers, map[string]any{"key": "Content-Type", "value": "application/json", "type": "text"})
		request["body"] = map[string]any{
			"mode":    "raw",
			"raw":     string(body),
			"options": map[string]any{"raw": map[string]any{"language": "json"}},
		}
	case len(op.FormParams) > 0 && op.Method != http.MethodGet:
		fields := []any{}
		for _, pair := range requestOf(server, op).form {
			fields = append(fields, map[string]any{"key": pair[0], "value": pair[1], "type": "text"})
		}
		request["body"] = map[string]any{"mode": "urlencoded", "urlencoded": fields}
	}
	if len(headers) > 0 {
		request["header"] = headers
	}

	return map[string]any{"name": op.Path, "request": request}
}

func collectionURL(host, path string, op *Operation) map[string]any {
	url := map[string]any{"host": []string{host}, "path": strings.Split(strings.TrimPrefix(path, "/"), "/")}
	raw := host + path

	if op != nil {
		var variables []any
		for _, param := range op.PathParams {
			variables = append(variables, map[string]any{"key": param.Name, "value": "", "description": param.Description})
		}
		if len(variables) > 0 {
			url["variable"] = variables
		}

		if op.Method == http.MethodGet && len(op.QueryParams) > 0 {
			var query []any
			var parts []string
			for _, param := range op.QueryParams {
				query = append(query, map[string]any{
					"key": param.Name, "value": "", "description": param.Description, "disabled": !param.Required,
				})
				if param.Required {
					parts = append(parts, param.Name+"=")
				}
			}
			url["query"] = query
			if len(parts) > 0 {
				raw += "?" + strings.Join(parts, "&")
			}
		}
	}

	url["raw"] = raw
	return url
}

func collectionAuth(op Operation) map[string]any {
	bearer := func(token string) map[string]any {
		return map[string]any{"type": "bearer", "bearer": []any{
			map[string]any{"key": "token", "value": "{{" + token + "}}", "type": "string"},
		}}
	}

	switch {
	case op.Access.Token == "admin":
		return bearer("admin_token")
	case op.Access.Token == "account":
		return bearer("access_token")
	case op.BasicAuth:
		return map[string]any{"type": "basic", "basic": []any{
			map[string]any{"key": "username", "value": "{{client_id}}", "type": "string"},
			map[string]any{"key": "password", "value": "{{client_secret}}", "type": "string"},
		}}
	}
	for _, h := range op.Headers {
		if strings.EqualFold(h.Name, "Authorization") {
			return bearer("access_token")
		}
	}
	return map[string]any{"type": "noauth"}
}

func sessionCookie(op Operation) string {
	switch {
	case op.Access.Token != "":
		return ""
	case op.Access.Session == "user":
		return brand.UserSessionCookie + "={{user_session}}"
	case op.Access.Session != "":
		return brand.AdminSessionCookie + "={{admin_session}}"
	}
	return ""
}
