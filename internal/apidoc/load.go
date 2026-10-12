package apidoc

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"loginer/i18n"
)

type loader struct {
	fset     *token.FileSet
	packages map[string]*packages.Package

	funcs    map[token.Pos]*site
	fields   map[token.Pos]*ast.Field
	types    map[token.Pos]*ast.CommentGroup
	problems map[*types.Var]Problem
}

type site struct {
	decl *ast.FuncDecl
	pkg  *packages.Package
}

func load(root string) (*loader, error) {
	cfg := &packages.Config{
		Dir: root,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
	}

	loaded, err := packages.Load(cfg, "./internal/...")
	if err != nil {
		return nil, fmt.Errorf("apidoc: loading the source: %w", err)
	}

	l := &loader{
		packages: map[string]*packages.Package{},
		funcs:    map[token.Pos]*site{},
		fields:   map[token.Pos]*ast.Field{},
		types:    map[token.Pos]*ast.CommentGroup{},
		problems: map[*types.Var]Problem{},
	}

	var failures []string
	for _, pkg := range loaded {
		for _, e := range pkg.Errors {
			failures = append(failures, e.Error())
		}
		l.packages[pkg.PkgPath] = pkg
		l.fset = pkg.Fset
	}
	if len(failures) > 0 {
		return nil, errors.New("apidoc: the source does not compile:\n" + strings.Join(failures, "\n"))
	}

	for _, pkg := range loaded {
		for _, file := range pkg.Syntax {
			l.indexDeclarations(file)
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					l.funcs[fn.Name.Pos()] = &site{decl: fn, pkg: pkg}
				}
			}
			l.indexProblems(pkg, file)
		}
	}

	return l, nil
}

func (l *loader) suffixed(suffix string) (*packages.Package, error) {
	for path, pkg := range l.packages {
		if strings.HasSuffix(path, suffix) {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("apidoc: no package %s was loaded", suffix)
}

func (l *loader) indexProblems(pkg *packages.Package, file *ast.File) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}

		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, value := range vs.Values {
				call, ok := value.(*ast.CallExpr)
				if !ok || len(call.Args) != 3 || calleeName(pkg.TypesInfo, call) != "respond.Define" {
					continue
				}

				code, _ := stringOf(pkg.TypesInfo, call.Args[1])
				problem := Problem{
					Status: statusOf(pkg.TypesInfo, call.Args[0]),
					Code:   code,
					Apps:   appsOf(call.Args[2]),
				}
				for _, app := range problem.Apps {
					if text, ok := i18n.Text(i18n.App(app), "error."+code); ok {
						problem.English = text
						break
					}
				}

				if v, ok := pkg.TypesInfo.Defs[vs.Names[i]].(*types.Var); ok {
					l.problems[v] = problem
				}
			}
		}
	}
}

func (l *loader) Problems() []Problem {
	out := make([]Problem, 0, len(l.problems))
	for _, problem := range l.problems {
		out = append(out, problem)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

func appsOf(expr ast.Expr) []string {
	name := ""
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		name = e.Sel.Name
	case *ast.Ident:
		name = e.Name
	}

	switch name {
	case "Public":
		return []string{string(i18n.ID)}
	case "Admin":
		return []string{string(i18n.Console)}
	default:
		return []string{string(i18n.ID), string(i18n.Console)}
	}
}

func calleeName(info *types.Info, call *ast.CallExpr) string {
	var ident *ast.Ident
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		ident = fun.Sel
	case *ast.Ident:
		ident = fun
	default:
		return ""
	}

	fn, ok := info.Uses[ident].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return ""
	}
	if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
		return ""
	}
	return fn.Pkg().Name() + "." + fn.Name()
}
