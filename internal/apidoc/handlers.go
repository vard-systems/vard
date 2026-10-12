package apidoc

import (
	"go/ast"
	"go/types"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

const maxCallDepth = 3

type handlerReading struct {
	route   route
	schemas *schemas
	seen    map[*ast.FuncDecl]bool

	op        Operation
	responses map[int]Response
	problems  map[string]Problem
	params    map[string]bool
}

func (l *loader) operation(r route, s *schemas) Operation {
	op := Operation{
		Method:        r.method,
		Path:          r.path,
		Handler:       r.handler.Pkg().Name() + "." + r.handler.Name(),
		Access:        r.guard.access,
		RateLimit:     r.guard.rateLimit,
		OriginChecked: r.guard.csrf && r.method != http.MethodGet,
		PathParams:    pathParams(r.path),
	}

	reading := &handlerReading{
		route:     r,
		schemas:   s,
		seen:      map[*ast.FuncDecl]bool{},
		op:        op,
		responses: map[int]Response{},
		problems:  map[string]Problem{},
		params:    map[string]bool{},
	}

	if site := l.funcs[r.handler.Pos()]; site != nil {
		reading.op.Summary, reading.op.Description = summary(r.handler.Name(), site.decl.Doc)
		l.read(reading, site, 0)
	}

	for _, response := range reading.responses {
		reading.op.Responses = append(reading.op.Responses, response)
	}
	sort.Slice(reading.op.Responses, func(i, j int) bool {
		a, b := reading.op.Responses[i].Status, reading.op.Responses[j].Status
		if a == 0 || b == 0 {
			return b == 0 && a != 0
		}
		return a < b
	})

	for _, list := range []*[]Param{&reading.op.QueryParams, &reading.op.FormParams} {
		for i, param := range *list {
			if param.Description == "" {
				(*list)[i].Description = protocolParams[param.Name]
			}
		}
	}

	for _, problem := range reading.problems {
		reading.op.Problems = append(reading.op.Problems, problem)
	}
	sort.Slice(reading.op.Problems, func(i, j int) bool {
		a, b := reading.op.Problems[i], reading.op.Problems[j]
		if a.Status != b.Status {
			return a.Status < b.Status
		}
		return a.Code < b.Code
	})

	return reading.op
}

func (l *loader) read(r *handlerReading, s *site, depth int) {
	if r.seen[s.decl] || s.decl.Body == nil {
		return
	}
	r.seen[s.decl] = true
	info := s.pkg.TypesInfo

	ast.Inspect(s.decl.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			if v, ok := info.Uses[n].(*types.Var); ok {
				if problem, ok := l.problems[v]; ok && problem.Status < http.StatusInternalServerError {
					r.problems[problem.Code] = problem
				}
			}

		case *ast.CallExpr:
			if l.context(r, s, n) {
				return true
			}
			fn := calledFunction(info, n)
			if fn == nil || fn.Pkg() != r.route.handler.Pkg() || depth >= maxCallDepth {
				return true
			}
			if callee := l.funcs[fn.Pos()]; callee != nil {
				l.read(r, callee, depth+1)
			}
		}
		return true
	})
}

func (l *loader) context(r *handlerReading, s *site, call *ast.CallExpr) bool {
	info := s.pkg.TypesInfo
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	receiver := typeName(info.TypeOf(sel.X))
	literal := func(i int) string {
		if i >= len(call.Args) {
			return ""
		}
		value, _ := stringOf(info, call.Args[i])
		return value
	}
	// A helper's doc comment is the best description of what it reads: the
	// function that reads a bearer token says where it looks.
	helperDoc := ""
	if s.decl.Name.Pos() != r.route.handler.Pos() {
		first, rest := summary(s.decl.Name.Name, s.decl.Doc)
		helperDoc = strings.TrimSpace(first + " " + rest)
	}

	switch receiver {
	case "github.com/gin-gonic/gin.Context":
		switch sel.Sel.Name {
		case "ShouldBindJSON", "BindJSON", "ShouldBind":
			if t := info.TypeOf(call.Args[0]); t != nil {
				if p, ok := t.(*types.Pointer); ok {
					t = p.Elem()
				}
				r.op.Body = r.schemas.forRequest(t)
			}
		case "ShouldBindQuery", "BindQuery":
			if t := info.TypeOf(call.Args[0]); t != nil {
				if p, ok := t.(*types.Pointer); ok {
					t = p.Elem()
				}
				r.queryStruct(t)
			}
		case "Query", "DefaultQuery", "GetQuery", "QueryArray":
			r.param(&r.op.QueryParams, "query", literal(0), "")
		case "PostForm", "DefaultPostForm", "GetPostForm":
			if r.route.method != http.MethodGet {
				r.param(&r.op.FormParams, "form", literal(0), "")
			}
		case "GetHeader":
			if name := literal(0); !strings.EqualFold(name, "Content-Type") {
				r.param(&r.op.Headers, "header", name, helperDoc)
			}
		case "JSON", "IndentedJSON", "PureJSON":
			r.respond(statusOf(info, call.Args[0]), "application/json", info.TypeOf(call.Args[1]), call.Args[1], info)
		case "Redirect":
			r.respond(statusOf(info, call.Args[0]), "", nil, nil, info)
		case "Status", "AbortWithStatus":
			r.respond(statusOf(info, call.Args[0]), "", nil, nil, info)
		case "Data":
			r.respond(statusOf(info, call.Args[0]), strings.SplitN(literal(1), ";", 2)[0], nil, nil, info)
		case "String":
			r.respond(statusOf(info, call.Args[0]), "text/plain", nil, nil, info)
		default:
			return false
		}
		return true

	case "net/http.Request":
		if sel.Sel.Name == "BasicAuth" {
			r.op.BasicAuth = true
			return true
		}

	case "net/url.Values":
		if sel.Sel.Name != "Get" {
			return false
		}
		fromBody := false
		if inner, ok := sel.X.(*ast.SelectorExpr); ok && inner.Sel.Name == "PostForm" {
			fromBody = true
		}
		switch {
		case r.route.method != http.MethodGet:
			r.param(&r.op.FormParams, "form", literal(0), "")
		case !fromBody:
			r.param(&r.op.QueryParams, "query", literal(0), "")
		}
		return true
	}

	return false
}

func (r *handlerReading) param(into *[]Param, where, name, description string) {
	if name == "" || r.params[where+":"+name] {
		return
	}
	r.params[where+":"+name] = true
	*into = append(*into, Param{Name: name, Description: description, Schema: &Schema{Type: "string"}})
}

func (r *handlerReading) queryStruct(t types.Type) {
	schema := r.schemas.forRequest(t)
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return
	}
	for i := range st.NumFields() {
		name := strings.Split(reflect.StructTag(st.Tag(i)).Get("form"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		param := Param{Name: name, Schema: &Schema{Type: "string"}}
		for _, property := range schema.Properties {
			if property.Name == st.Field(i).Name() || property.Name == name {
				param.Schema, param.Required, param.Description = property.Schema, property.Required, property.Description
				if len(property.Rules) > 0 {
					param.Description = strings.TrimSpace(param.Description + " " + exported(strings.Join(property.Rules, ", ")) + ".")
				}
			}
		}
		if !r.params["query:"+name] {
			r.params["query:"+name] = true
			r.op.QueryParams = append(r.op.QueryParams, param)
		}
	}
}

func (r *handlerReading) respond(status int, contentType string, t types.Type, expr ast.Expr, info *types.Info) {
	if status >= http.StatusInternalServerError {
		return
	}
	if _, taken := r.responses[status]; taken {
		return
	}

	response := Response{Status: status, ContentType: contentType}
	if t != nil {
		if lit, ok := expr.(*ast.CompositeLit); ok && isGinH(t) {
			response.Schema = r.object(lit, info)
		} else {
			response.Schema = r.schemas.forResponse(t)
		}
	}
	r.responses[status] = response
}

func (r *handlerReading) object(lit *ast.CompositeLit, info *types.Info) *Schema {
	schema := &Schema{Type: "object"}
	for _, element := range lit.Elts {
		kv, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := stringOf(info, kv.Key)
		if !ok {
			continue
		}

		value := &Schema{}
		if nested, ok := kv.Value.(*ast.CompositeLit); ok && isGinH(info.TypeOf(nested)) {
			value = r.object(nested, info)
		} else if t := info.TypeOf(kv.Value); t != nil {
			value = r.schemas.forResponse(t)
		}
		schema.Properties = append(schema.Properties, Property{Name: key, Schema: value, Required: !value.Nullable})
	}
	return schema
}

func isGinH(t types.Type) bool {
	return typeName(t) == "github.com/gin-gonic/gin.H"
}

func typeName(t types.Type) string {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	if alias, ok := t.(*types.Alias); ok {
		obj := alias.Obj()
		if obj.Pkg() != nil {
			return obj.Pkg().Path() + "." + obj.Name()
		}
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return ""
	}
	return named.Obj().Pkg().Path() + "." + named.Obj().Name()
}

func calledFunction(info *types.Info, call *ast.CallExpr) *types.Func {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		if selection := info.Selections[fun]; selection != nil {
			fn, _ := selection.Obj().(*types.Func)
			return fn
		}
		fn, _ := info.Uses[fun.Sel].(*types.Func)
		return fn
	case *ast.Ident:
		fn, _ := info.Uses[fun].(*types.Func)
		return fn
	}
	return nil
}

func summary(name string, doc *ast.CommentGroup) (string, string) {
	if doc == nil {
		return "", ""
	}

	text := sentence(doc.Text())
	if rest, ok := strings.CutPrefix(text, name+" "); ok {
		text = exported(rest)
	}

	first, rest := text, ""
	if end := sentenceEnd(text); end > 0 {
		first, rest = text[:end], strings.TrimSpace(text[end:])
	}

	return first, rest
}

func sentenceEnd(text string) int {
	for i := 0; i < len(text)-1; i++ {
		if text[i] != '.' {
			continue
		}
		next := text[i+1]
		if next == '\n' || (next == ' ' && i+2 < len(text) && isUpper(text[i+2])) {
			return i + 1
		}
	}
	return 0
}

func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }
