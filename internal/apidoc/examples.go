package apidoc

import (
	"encoding/json"
	"fmt"
	"go/format"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"loginer/internal/brand"
	"loginer/internal/model"
)

type sample struct {
	title, language, code string
}

type request struct {
	method   string
	base     string
	segments []segment
	query    [][2]string
	form     [][2]string
	body     any
	session  string
	basic    bool
	bearer   bool
	token    string
}

type segment struct {
	text  string
	param bool
}

func requestOf(server Server, op Operation) request {
	r := request{method: op.Method}

	if server.Key == "admin" {
		r.base = "ADMIN_API"
	} else {
		r.base = "ISSUER"
	}

	literal := ""
	for part := range strings.SplitSeq(strings.TrimPrefix(op.Path, "/"), "/") {
		if name, ok := strings.CutPrefix(part, ":"); ok {
			r.segments = append(r.segments, segment{text: literal + "/"}, segment{text: name, param: true})
			literal = ""
			continue
		}
		literal += "/" + part
	}
	if literal != "" {
		r.segments = append(r.segments, segment{text: literal})
	}

	if op.Method == http.MethodGet {
		for _, param := range op.QueryParams {
			if param.Required || len(r.query) < 3 {
				r.query = append(r.query, [2]string{param.Name, exampleValue(param.Name)})
			}
		}
	}

	r.token = "ACCESS_TOKEN"
	switch {
	case op.Access.Token != "":
		r.bearer = true
		if op.Access.Token == "admin" {
			r.token = "ADMIN_TOKEN"
		}
	case op.Access.Session == "user":
		r.session = brand.UserSessionCookie
	case op.Access.Session != "":
		r.session = brand.AdminSessionCookie
	case op.BasicAuth:
		r.basic = true
	}
	for _, h := range op.Headers {
		if strings.EqualFold(h.Name, "Authorization") && !op.BasicAuth {
			r.bearer = true
		}
	}

	if op.Body != nil {
		r.body = example(op.Body, "", 0, true)
	} else if op.Method != http.MethodGet {
		for _, param := range op.FormParams {
			if r.basic && (param.Name == "client_id" || param.Name == "client_secret") {
				continue
			}
			if r.bearer && param.Name == "access_token" {
				continue
			}
			r.form = append(r.form, [2]string{param.Name, exampleValue(param.Name)})
		}
		r.form = oneGrant(r.form)
	}

	return r
}

var codeExchange = map[string]bool{"grant_type": true, "code": true, "redirect_uri": true, "code_verifier": true}

func oneGrant(form [][2]string) [][2]string {
	isToken := false
	for _, pair := range form {
		isToken = isToken || pair[0] == "grant_type"
	}
	if !isToken {
		return form
	}

	var out [][2]string
	for _, pair := range form {
		if codeExchange[pair[0]] {
			out = append(out, pair)
		}
	}
	return out
}

func exampleValue(name string) string {
	switch name {
	case "grant_type":
		return model.GrantAuthorizationCode
	case "response_type":
		return "code"
	case "code_challenge_method":
		return model.PKCES256
	case "code":
		return "SplxlOBeZQQYbYS6WxSbIA"
	case "state":
		return "af0ifjsldkj"
	case "nonce":
		return "n-0S6_WzA2Mj"
	case "prompt":
		return "login"
	case "audience":
		return "https://api.example.com"
	}
	if value := exampleString(&Schema{Type: "string"}, name); value != "string" {
		return value
	}
	return "value"
}

func (r request) samples() []sample {
	return []sample{
		{"cURL", "bash", r.curl()},
		{"Java", "java", r.java()},
		{"Go", "go", r.golang()},
		{"Python", "python", r.python()},
	}
}

func (r request) curl() string {
	var target strings.Builder
	target.WriteString("$" + r.base)
	for _, s := range r.segments {
		if s.param {
			target.WriteString("$" + strings.ToUpper(s.text))
		} else {
			target.WriteString(s.text)
		}
	}
	if len(r.query) > 0 {
		target.WriteString("?" + encode(r.query))
	}

	lines := []string{"curl"}
	if r.method != http.MethodGet {
		lines[0] += " -X " + r.method
	}
	lines[0] += ` "` + target.String() + `"`

	switch {
	case r.session != "":
		lines = append(lines, `-b "`+r.session+`=$SESSION"`)
	case r.basic:
		lines = append(lines, `-u "$CLIENT_ID:$CLIENT_SECRET"`)
	}
	if r.bearer {
		lines = append(lines, `-H "Authorization: Bearer $`+r.token+`"`)
	}

	switch {
	case r.body != nil:
		body, _ := json.Marshal(r.body)
		quoted := strings.ReplaceAll(string(body), "'", `'\''`)
		lines = append(lines, `-H "Content-Type: application/json"`, "-d '"+quoted+"'")
	case len(r.form) > 0:
		for _, pair := range r.form {
			lines = append(lines, `-d "`+encode([][2]string{pair})+`"`)
		}
	}

	return strings.Join(lines, " \\\n  ")
}

func (r request) java() string {
	var b strings.Builder

	var uri strings.Builder
	uri.WriteString(r.base)
	for _, s := range r.segments {
		if s.param {
			uri.WriteString(" + " + camel(s.text, false))
		} else {
			uri.WriteString(" + " + quoteJava(s.text))
		}
	}
	if len(r.query) > 0 {
		uri.WriteString(" + " + quoteJava("?"+encode(r.query)))
	}

	b.WriteString("HttpRequest request = HttpRequest.newBuilder()\n")
	fmt.Fprintf(&b, "        .uri(URI.create(%s))\n", uri.String())

	switch {
	case r.session != "":
		fmt.Fprintf(&b, "        .header(\"Cookie\", \"%s=\" + SESSION)\n", r.session)
	case r.basic:
		b.WriteString("        .header(\"Authorization\", \"Basic \" + Base64.getEncoder()\n")
		b.WriteString("                .encodeToString((CLIENT_ID + \":\" + CLIENT_SECRET).getBytes(UTF_8)))\n")
	}
	if r.bearer {
		fmt.Fprintf(&b, "        .header(\"Authorization\", \"Bearer \" + %s)\n", r.token)
	}

	publisher := "BodyPublishers.noBody()"
	switch {
	case r.body != nil:
		b.WriteString("        .header(\"Content-Type\", \"application/json\")\n")
		indented, _ := json.MarshalIndent(r.body, "                ", "  ")
		publisher = "BodyPublishers.ofString(\"\"\"\n                " + string(indented) + "\n                \"\"\")"
	case len(r.form) > 0:
		b.WriteString("        .header(\"Content-Type\", \"application/x-www-form-urlencoded\")\n")
		lines := make([]string, len(r.form))
		for i, pair := range r.form {
			prefix := "&"
			if i == 0 {
				prefix = ""
			}
			lines[i] = quoteJava(prefix + encode([][2]string{pair}))
		}
		publisher = "BodyPublishers.ofString(\n                " +
			strings.Join(lines, "\n                + ") + ")"
	}

	switch r.method {
	case http.MethodGet:
		b.WriteString("        .GET()\n")
	case http.MethodDelete:
		b.WriteString("        .DELETE()\n")
	case http.MethodPost, http.MethodPut:
		fmt.Fprintf(&b, "        .%s(%s)\n", r.method, publisher)
	default:
		fmt.Fprintf(&b, "        .method(%q, %s)\n", r.method, publisher)
	}
	b.WriteString("        .build();\n\n")

	b.WriteString("HttpResponse<String> response = HttpClient.newHttpClient()\n")
	b.WriteString("        .send(request, BodyHandlers.ofString());")

	return b.String()
}

func (r request) golang() string {
	var b strings.Builder

	var target strings.Builder
	target.WriteString(goName(r.base))
	for _, s := range r.segments {
		if s.param {
			target.WriteString("+" + camel(s.text, true))
		} else {
			target.WriteString("+" + fmt.Sprintf("%q", s.text))
		}
	}
	if len(r.query) > 0 {
		target.WriteString("+" + fmt.Sprintf("%q", "?"+encode(r.query)))
	}

	body := "nil"
	switch {
	case r.body != nil:
		indented, _ := json.MarshalIndent(r.body, "", "\t")
		fmt.Fprintf(&b, "body := strings.NewReader(`%s`)\n", indented)
		body = "body"
	case len(r.form) > 0:
		b.WriteString("form := url.Values{\n")
		for _, pair := range r.form {
			fmt.Fprintf(&b, "\t%q: {%q},\n", pair[0], pair[1])
		}
		b.WriteString("}\n")
		body = "strings.NewReader(form.Encode())"
	}

	fmt.Fprintf(&b, "req, err := http.NewRequest(http.Method%s, %s, %s)\n", goMethod(r.method), target.String(), body)
	b.WriteString("if err != nil {\n\treturn err\n}\n")

	switch {
	case r.body != nil:
		b.WriteString("req.Header.Set(\"Content-Type\", \"application/json\")\n")
	case len(r.form) > 0:
		b.WriteString("req.Header.Set(\"Content-Type\", \"application/x-www-form-urlencoded\")\n")
	}
	switch {
	case r.session != "":
		fmt.Fprintf(&b, "req.AddCookie(&http.Cookie{Name: %q, Value: session})\n", r.session)
	case r.basic:
		b.WriteString("req.SetBasicAuth(clientID, clientSecret)\n")
	}
	if r.bearer {
		fmt.Fprintf(&b, "req.Header.Set(\"Authorization\", \"Bearer \"+%s)\n", goName(r.token))
	}

	b.WriteString("\nres, err := http.DefaultClient.Do(req)\n")
	b.WriteString("if err != nil {\n\treturn err\n}\n")
	b.WriteString("defer res.Body.Close()")

	if formatted, err := format.Source([]byte(b.String())); err == nil {
		return string(formatted)
	}
	return b.String()
}

func goMethod(method string) string {
	return method[:1] + strings.ToLower(method[1:])
}

func (r request) python() string {
	var b strings.Builder

	var target strings.Builder
	target.WriteString("{" + r.base + "}")
	for _, s := range r.segments {
		if s.param {
			target.WriteString("{" + s.text + "}")
		} else {
			target.WriteString(s.text)
		}
	}

	args := []string{`f"` + target.String() + `"`}
	if len(r.query) > 0 {
		args = append(args, "params="+pythonPairs(r.query))
	}
	switch {
	case r.body != nil:
		args = append(args, "json="+pythonLiteral(r.body, 1))
	case len(r.form) > 0:
		args = append(args, "data="+pythonPairs(r.form))
	}
	switch {
	case r.session != "":
		args = append(args, fmt.Sprintf("cookies={%q: SESSION}", r.session))
	case r.basic:
		args = append(args, "auth=(CLIENT_ID, CLIENT_SECRET)")
	}
	if r.bearer {
		args = append(args, `headers={"Authorization": f"Bearer {`+r.token+`}"}`)
	}

	call := "requests." + strings.ToLower(r.method)
	if len(args) == 1 {
		fmt.Fprintf(&b, "response = %s(%s)\n", call, args[0])
	} else {
		fmt.Fprintf(&b, "response = %s(\n", call)
		for _, arg := range args {
			fmt.Fprintf(&b, "    %s,\n", arg)
		}
		b.WriteString(")\n")
	}
	b.WriteString("response.raise_for_status()")

	return b.String()
}

func pythonPairs(pairs [][2]string) string {
	var b strings.Builder
	b.WriteString("{\n")
	for _, pair := range pairs {
		fmt.Fprintf(&b, "        %s: %s,\n", pythonString(pair[0]), pythonString(pair[1]))
	}
	b.WriteString("    }")
	return b.String()
}

func pythonLiteral(value any, depth int) string {
	pad := strings.Repeat("    ", depth)
	inner := pad + "    "

	switch v := value.(type) {
	case orderedObject:
		if len(v) == 0 {
			return "{}"
		}
		var b strings.Builder
		b.WriteString("{\n")
		for _, f := range v {
			fmt.Fprintf(&b, "%s%s: %s,\n", inner, pythonString(f.name), pythonLiteral(f.value, depth+1))
		}
		b.WriteString(pad + "}")
		return b.String()
	case map[string]any:
		if len(v) == 0 {
			return "{}"
		}
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("{\n")
		for _, key := range keys {
			fmt.Fprintf(&b, "%s%s: %s,\n", inner, pythonString(key), pythonLiteral(v[key], depth+1))
		}
		b.WriteString(pad + "}")
		return b.String()
	case []any:
		if len(v) == 0 {
			return "[]"
		}
		var b strings.Builder
		b.WriteString("[\n")
		for _, item := range v {
			fmt.Fprintf(&b, "%s%s,\n", inner, pythonLiteral(item, depth+1))
		}
		b.WriteString(pad + "]")
		return b.String()
	case string:
		return pythonString(v)
	case bool:
		if v {
			return "True"
		}
		return "False"
	case nil:
		return "None"
	default:
		return fmt.Sprint(v)
	}
}

func pythonString(s string) string {
	quoted, _ := json.Marshal(s)
	return string(quoted)
}

func quoteJava(s string) string {
	return fmt.Sprintf("%q", s)
}

func encode(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, pair := range pairs {
		parts[i] = url.QueryEscape(pair[0]) + "=" + url.QueryEscape(pair[1])
	}
	return strings.Join(parts, "&")
}

func goName(name string) string {
	return camel(strings.ToLower(name), true)
}

func camel(name string, goStyle bool) string {
	parts := strings.Split(name, "_")
	for i := 1; i < len(parts); i++ {
		switch upper := strings.ToUpper(parts[i]); {
		case goStyle && (upper == "ID" || upper == "URL" || upper == "URI" || upper == "API"):
			parts[i] = upper
		default:
			parts[i] = exported(parts[i])
		}
	}
	return strings.Join(parts, "")
}
