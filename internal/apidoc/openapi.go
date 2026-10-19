package apidoc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"loginer/internal/brand"
	"loginer/internal/model"
)

func (b *built) openAPI(server Server) ([]byte, error) {
	s := b.schemas[server.Key]

	tags := []map[string]any{}
	paths := map[string]map[string]any{}
	ids := map[string]int{}

	for _, section := range server.Sections {
		tags = append(tags, map[string]any{"name": section.Title, "description": section.Intro})

		for _, op := range section.Operations {
			path := openAPIPath(op.Path)
			if paths[path] == nil {
				paths[path] = map[string]any{}
			}

			id := operationID(op)
			ids[id]++
			if ids[id] > 1 {
				id += exported(strings.ToLower(op.Method))
			}

			paths[path][strings.ToLower(op.Method)] = b.openAPIOperation(op, section, id)
		}
	}

	components := map[string]any{}
	for _, schema := range s.Components() {
		components[schema.Name] = jsonSchema(schema, true)
	}
	components["Problem"] = map[string]any{
		"type":        "object",
		"description": "An error the API answers with. `code` is stable, and what an application should act on; `error` is its English sentence; `params` fills in the sentence's {placeholders}.",
		"required":    []string{"error", "code"},
		"properties": map[string]any{
			"error":  map[string]any{"type": "string"},
			"code":   map[string]any{"type": "string"},
			"params": map[string]any{"type": "object", "additionalProperties": true},
		},
	}
	if server.Key == "public" {
		components["OAuthError"] = map[string]any{
			"type":        "object",
			"description": "An OAuth 2.0 error (RFC 6749 section 5.2), which is how the provider's own endpoints answer.",
			"required":    []string{"error"},
			"properties": map[string]any{
				"error":             map[string]any{"type": "string"},
				"error_description": map[string]any{"type": "string"},
			},
		}
	}

	document := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       brand.Name + " " + server.Title,
			"version":     "1",
			"description": serverDescription(server.Key),
		},
		"tags":  tags,
		"paths": paths,
		"components": map[string]any{
			"schemas":         components,
			"securitySchemes": securitySchemes(server.Key),
		},
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

func serverDescription(key string) string {
	if key == "admin" {
		return "The admin API the panel calls, with an administrator's session. It is served on its own listener, meant to be reachable only from where administrators work."
	}
	return "The OAuth 2.0 and OpenID Connect provider applications sign their users in with, and the account API the sign-in pages call."
}

func securitySchemes(key string) map[string]any {
	if key == "admin" {
		return map[string]any{
			"adminSession": map[string]any{
				"type": "apiKey", "in": "cookie", "name": brand.AdminSessionCookie,
				"description": "An administrator's session, set by signing in.",
			},
			"adminToken": map[string]any{
				"type": "http", "scheme": "bearer", "bearerFormat": "JWT",
				"description": "An access token for the admin API (audience " + brand.AdminAPIIdentifier + "), from the client credentials grant — admin-cli's, or another application's an administrator authorized. Its scopes are the permissions the route checks.",
			},
		}
	}
	return map[string]any{
		"userSession": map[string]any{
			"type": "apiKey", "in": "cookie", "name": brand.UserSessionCookie,
			"description": "A user's session at the provider, set by signing in on the sign-in pages.",
		},
		"clientSecretBasic": map[string]any{
			"type": "http", "scheme": "basic",
			"description": "An application's client_id and client_secret, form-encoded and then base64-encoded (RFC 6749 section 2.3.1). The same credentials may be sent in the form instead.",
		},
		"accessToken": map[string]any{
			"type": "http", "scheme": "bearer", "bearerFormat": "JWT",
			"description": "An access token this server issued.",
		},
		"accountToken": map[string]any{
			"type": "http", "scheme": "bearer", "bearerFormat": "JWT",
			"description": "A user's access token for the account API (audience " + brand.AccountAPIIdentifier + "), with account.read to read and account.write to change.",
		},
	}
}

func operationID(op Operation) string {
	pkg, method, _ := strings.Cut(op.Handler, ".")
	return pkg + method
}

func (b *built) openAPIOperation(op Operation, section Section, id string) map[string]any {
	out := map[string]any{
		"operationId": id,
		"tags":        []string{section.Title},
		"summary":     strings.TrimSuffix(op.Summary, "."),
	}

	if op.Description != "" {
		out["description"] = op.Description
	}

	var parameters []map[string]any
	add := func(in string, params []Param) {
		for _, p := range params {
			parameter := map[string]any{"name": p.Name, "in": in, "schema": jsonSchema(p.Schema, false)}
			if p.Required || in == "path" {
				parameter["required"] = true
			}
			if p.Description != "" {
				parameter["description"] = p.Description
			}
			parameters = append(parameters, parameter)
		}
	}
	add("path", op.PathParams)
	add("query", op.QueryParams)

	bearer := false
	var headers []Param
	for _, h := range op.Headers {
		if strings.EqualFold(h.Name, "Authorization") {
			bearer = !op.BasicAuth
			continue
		}
		headers = append(headers, h)
	}
	add("header", headers)
	if len(parameters) > 0 {
		out["parameters"] = parameters
	}

	switch {
	case op.Body != nil:
		out["requestBody"] = map[string]any{
			"required": true,
			"content":  map[string]any{"application/json": map[string]any{"schema": jsonSchema(op.Body, false)}},
		}
	case len(op.FormParams) > 0:
		properties := map[string]any{}
		for _, p := range op.FormParams {
			properties[p.Name] = jsonSchema(p.Schema, false)
		}
		out["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{"application/x-www-form-urlencoded": map[string]any{
				"schema": map[string]any{"type": "object", "properties": properties},
			}},
		}
	}

	switch {
	case op.Access.Session == "user" && op.Access.Token == "account":
		out["security"] = []map[string]any{{"userSession": []string{}}, {"accountToken": []string{accountScope(op)}}}
	case op.Access.Session == "user":
		out["security"] = []map[string]any{{"userSession": []string{}}}
	case op.Access.Token == "admin":
		out["security"] = []map[string]any{{"adminSession": []string{}}, {"adminToken": op.Access.Permissions}}
	case op.Access.Session != "":
		out["security"] = []map[string]any{{"adminSession": []string{}}}
	case op.BasicAuth:
		out["security"] = []map[string]any{{"clientSecretBasic": []string{}}, {}}
	case bearer:
		out["security"] = []map[string]any{{"accessToken": []string{}}}
	default:
		out["security"] = []map[string]any{}
	}

	if len(op.Access.Permissions) > 0 {
		out["x-permissions"] = op.Access.Permissions
		if op.Access.Anywhere {
			out["x-permission-scope"] = "any-application"
		}
	}
	if op.Access.SuperAdmin {
		out["x-super-admin"] = true
	}
	if op.RateLimit != "" {
		out["x-rate-limit"] = op.RateLimit
	}

	out["responses"] = openAPIResponses(op, section)

	return out
}

func openAPIResponses(op Operation, section Section) map[string]any {
	responses := map[string]any{}

	for _, r := range op.Responses {
		key := strconv.Itoa(r.Status)
		description := http.StatusText(r.Status)
		if r.Status == 0 {
			key, description = "default", "An error"
		}
		if r.Status >= 300 && r.Status < 400 {
			description = "Redirects the browser: see the Location header."
		}

		response := map[string]any{"description": description}
		if r.ContentType != "" {
			media := map[string]any{}
			if r.Schema != nil {
				media["schema"] = jsonSchema(r.Schema, false)
			}
			if r.Status == 0 && section.Slug == "oauth" {
				media["schema"] = map[string]any{"$ref": "#/components/schemas/OAuthError"}
			}
			response["content"] = map[string]any{r.ContentType: media}
		}
		responses[key] = response
	}

	byStatus := map[int][]Problem{}
	for _, p := range op.Problems {
		byStatus[p.Status] = append(byStatus[p.Status], p)
	}
	for status, problems := range byStatus {
		key := strconv.Itoa(status)
		if _, taken := responses[key]; taken {
			continue
		}

		codes := make([]string, len(problems))
		examples := map[string]any{}
		for i, p := range problems {
			codes[i] = "`" + p.Code + "`"
			examples[p.Code] = map[string]any{
				"summary": p.English,
				"value":   map[string]any{"error": p.English, "code": p.Code},
			}
		}

		responses[key] = map[string]any{
			"description": http.StatusText(status) + ": " + strings.Join(codes, ", "),
			"content": map[string]any{"application/json": map[string]any{
				"schema":   map[string]any{"$ref": "#/components/schemas/Problem"},
				"examples": examples,
			}},
		}
	}

	if len(responses) == 0 {
		responses["default"] = map[string]any{"description": "See the description."}
	}
	return responses
}

func jsonSchema(s *Schema, definition bool) map[string]any {
	if s == nil {
		return map[string]any{}
	}

	if s.Name != "" && !definition {
		ref := map[string]any{"$ref": "#/components/schemas/" + s.Name}
		if s.Nullable {
			return map[string]any{"anyOf": []any{ref, map[string]any{"type": "null"}}}
		}
		return ref
	}

	out := map[string]any{}
	if s.Type != "" {
		if s.Nullable {
			out["type"] = []string{s.Type, "null"}
		} else {
			out["type"] = s.Type
		}
	}
	if s.Format != "" {
		out["format"] = s.Format
	}
	if s.Description != "" {
		out["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		out["enum"] = s.Enum
	}
	if s.Items != nil {
		out["items"] = jsonSchema(s.Items, false)
	}
	if s.Values != nil {
		out["additionalProperties"] = jsonSchema(s.Values, false)
	}

	if s.Type == "object" && s.Values == nil {
		properties := map[string]any{}
		var required []string
		for _, p := range s.Properties {
			property := jsonSchema(p.Schema, false)
			description := p.Description
			if len(p.Rules) > 0 {
				description = strings.TrimSpace(description + " " + exported(strings.Join(p.Rules, ", ")) + ".")
			}
			if description != "" {
				if _, isRef := property["$ref"]; isRef {
					property = map[string]any{"allOf": []any{property}, "description": description}
				} else {
					property["description"] = description
				}
			}
			properties[p.Name] = property
			if p.Required {
				required = append(required, p.Name)
			}
		}
		out["properties"] = properties
		if len(required) > 0 {
			out["required"] = required
		}
	}

	return out
}

func accountScope(op Operation) string {
	if op.Method == http.MethodGet {
		return model.ScopeAccountRead
	}
	return model.ScopeAccountWrite
}
