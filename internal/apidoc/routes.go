package apidoc

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

type route struct {
	method  string
	path    string
	guard   guard
	handler *types.Func
}

type guard struct {
	access    Access
	rateLimit string
	csrf      bool
}

func (g guard) with(other guard) guard {
	out := g
	out.access.Permissions = slices.Clone(g.access.Permissions)
	if other.access.Session != "" {
		out.access.Session = other.access.Session
		out.access.Token = other.access.Token
	}
	out.access.Permissions = append(out.access.Permissions, other.access.Permissions...)
	out.access.Anywhere = out.access.Anywhere || other.access.Anywhere
	out.access.SuperAdmin = out.access.SuperAdmin || other.access.SuperAdmin
	if other.rateLimit != "" {
		out.rateLimit = other.rateLimit
	}
	out.csrf = out.csrf || other.csrf
	return out
}

var methods = map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}

// table reads the route table in server.go. An unrecognised guard stops it, so
// a new kind of guard cannot go undocumented.
type table struct {
	pkg    *packages.Package
	groups map[types.Object]guardedPrefix
	routes []route
}

type guardedPrefix struct {
	prefix string
	guard  guard
}

func (l *loader) routes(function string) ([]route, error) {
	pkg, err := l.suffixed("/internal/api")
	if err != nil {
		return nil, err
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != function {
				continue
			}

			t := &table{pkg: pkg, groups: map[types.Object]guardedPrefix{}}

			engine := fn.Type.Params.List[0].Names[0]
			t.groups[pkg.TypesInfo.Defs[engine]] = guardedPrefix{}

			if err := t.block(fn.Body.List); err != nil {
				return nil, fmt.Errorf("apidoc: %s: %w", function, err)
			}
			return t.routes, nil
		}
	}

	return nil, fmt.Errorf("apidoc: there is no %s in %s", function, pkg.PkgPath)
}

func (t *table) block(statements []ast.Stmt) error {
	for _, statement := range statements {
		switch s := statement.(type) {
		case *ast.BlockStmt:
			if err := t.block(s.List); err != nil {
				return err
			}
		case *ast.AssignStmt:
			if err := t.group(s); err != nil {
				return err
			}
		case *ast.ExprStmt:
			if err := t.route(s); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *table) group(s *ast.AssignStmt) error {
	if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
		return nil
	}
	call, ok := s.Rhs[0].(*ast.CallExpr)
	if !ok {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Group" {
		return nil
	}
	parent, ok := t.parent(sel.X)
	if !ok {
		return nil
	}

	path, ok := stringOf(t.pkg.TypesInfo, call.Args[0])
	if !ok {
		return fmt.Errorf("a group's path is not a constant at %s", t.pkg.Fset.Position(call.Pos()))
	}

	g := parent.guard
	for _, middleware := range call.Args[1:] {
		guard, err := t.guard(middleware)
		if err != nil {
			return err
		}
		g = g.with(guard)
	}

	name := s.Lhs[0].(*ast.Ident)
	object := t.pkg.TypesInfo.Defs[name]
	if object == nil {
		object = t.pkg.TypesInfo.Uses[name]
	}
	t.groups[object] = guardedPrefix{prefix: parent.prefix + path, guard: g}

	return nil
}

func (t *table) route(s *ast.ExprStmt) error {
	call, ok := s.X.(*ast.CallExpr)
	if !ok {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !methods[sel.Sel.Name] {
		return nil
	}
	parent, ok := t.parent(sel.X)
	if !ok {
		return nil
	}

	where := t.pkg.Fset.Position(call.Pos())
	path, ok := stringOf(t.pkg.TypesInfo, call.Args[0])
	if !ok {
		return fmt.Errorf("a route's path is not a constant at %s", where)
	}
	if len(call.Args) < 2 {
		return fmt.Errorf("a route has no handler at %s", where)
	}

	g := parent.guard
	for _, middleware := range call.Args[1 : len(call.Args)-1] {
		guard, err := t.guard(middleware)
		if err != nil {
			return err
		}
		g = g.with(guard)
	}

	handler := t.function(call.Args[len(call.Args)-1])
	if handler == nil {
		return fmt.Errorf("the handler at %s is not a function or a method", where)
	}

	t.routes = append(t.routes, route{
		method:  sel.Sel.Name,
		path:    parent.prefix + path,
		guard:   g,
		handler: handler,
	})

	return nil
}

func (t *table) parent(expr ast.Expr) (guardedPrefix, bool) {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return guardedPrefix{}, false
	}
	g, ok := t.groups[t.pkg.TypesInfo.Uses[ident]]
	return g, ok
}

func (t *table) guard(expr ast.Expr) (guard, error) {
	info := t.pkg.TypesInfo
	where := t.pkg.Fset.Position(expr.Pos())

	if call, ok := expr.(*ast.CallExpr); ok {
		constants := func() []string {
			var out []string
			for _, arg := range call.Args {
				if value, ok := stringOf(info, arg); ok {
					out = append(out, value)
				}
			}
			return out
		}

		switch calleeName(info, call) {
		case "session.Require":
			return guard{access: Access{Session: "admin"}}, nil
		case "session.RequireAny":
			return guard{access: Access{Session: "admin", Token: "admin"}}, nil
		case "session.RequireSetup":
			return guard{access: Access{Session: "admin-setup"}}, nil
		case "session.Can":
			return guard{access: Access{Permissions: constants()}}, nil
		case "session.CanAnywhere":
			return guard{access: Access{Permissions: constants(), Anywhere: true}}, nil
		case "session.RequireSuperAdmin":
			return guard{access: Access{SuperAdmin: true}}, nil
		}
		return guard{}, fmt.Errorf("the middleware at %s is not one the reference knows; teach internal/apidoc/routes.go what it means", where)
	}

	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if selection := info.Selections[sel]; selection != nil {
			switch selection.Obj().Name() {
			case "limit":
				return guard{rateLimit: "sign-in"}, nil
			case "tokens":
				return guard{rateLimit: "token"}, nil
			case "csrf":
				return guard{csrf: true}, nil
			case "RequireSession":
				return guard{access: Access{Session: "user", Token: "account"}}, nil
			}
		}
	}

	return guard{}, fmt.Errorf("the middleware at %s is not one the reference knows; teach internal/apidoc/routes.go what it means", where)
}

func (t *table) function(expr ast.Expr) *types.Func {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		if selection := t.pkg.TypesInfo.Selections[e]; selection != nil {
			fn, _ := selection.Obj().(*types.Func)
			return fn
		}
		fn, _ := t.pkg.TypesInfo.Uses[e.Sel].(*types.Func)
		return fn
	case *ast.Ident:
		fn, _ := t.pkg.TypesInfo.Uses[e].(*types.Func)
		return fn
	}
	return nil
}

func openAPIPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if name, ok := strings.CutPrefix(segment, ":"); ok {
			segments[i] = "{" + name + "}"
		}
	}
	return strings.Join(segments, "/")
}

func pathParams(path string) []Param {
	var out []Param
	for segment := range strings.SplitSeq(path, "/") {
		if name, ok := strings.CutPrefix(segment, ":"); ok {
			out = append(out, Param{Name: name, Required: true, Schema: &Schema{Type: "string"}})
		}
	}
	return out
}
