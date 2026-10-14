package apidoc

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Schema struct {
	Type        string
	Format      string
	Name        string
	Description string
	Nullable    bool
	Enum        []string
	Properties  []Property
	Items       *Schema
	Values      *Schema
	Of          *Schema
}

func (s *Schema) resolved() *Schema {
	if s != nil && s.Of != nil {
		return s.Of
	}
	return s
}

type Property struct {
	Name        string
	Schema      *Schema
	Required    bool
	Rules       []string
	Description string
}

type schemas struct {
	loader  *loader
	named   map[string]*Schema
	seen    map[*types.TypeName]string
	structs map[structKey]*Schema
}

type structKey struct {
	name    *types.TypeName
	request bool
}

func newSchemas(l *loader) *schemas {
	return &schemas{
		loader:  l,
		named:   map[string]*Schema{},
		seen:    map[*types.TypeName]string{},
		structs: map[structKey]*Schema{},
	}
}

func (s *schemas) forRequest(t types.Type) *Schema {
	return s.of(t, true, 0)
}

func (s *schemas) forResponse(t types.Type) *Schema {
	return s.of(t, false, 0)
}

const maxDepth = 12

func (s *schemas) of(t types.Type, request bool, depth int) *Schema {
	if depth > maxDepth {
		return &Schema{}
	}

	if named, ok := t.(*types.Named); ok {
		if known := wellKnown(named); known != nil {
			return known
		}
	}

	switch u := t.Underlying().(type) {
	case *types.Basic:
		schema := basic(u)
		if named, ok := t.(*types.Named); ok {
			schema.Enum = s.loader.enum(named)
		}
		return schema

	case *types.Pointer:
		elem := s.of(u.Elem(), request, depth)
		// A named struct's schema may still be being built — it may be the
		// very one this pointer is inside — so it is referred to, not copied.
		if elem.Name != "" {
			return &Schema{Name: elem.Name, Type: elem.Type, Nullable: true, Of: elem}
		}
		copied := *elem
		copied.Nullable = true
		return &copied

	case *types.Slice:
		if b, ok := u.Elem().(*types.Basic); ok && b.Kind() == types.Byte {
			return &Schema{Type: "string", Format: "byte"}
		}
		return &Schema{Type: "array", Items: s.of(u.Elem(), request, depth+1)}

	case *types.Array:
		return &Schema{Type: "array", Items: s.of(u.Elem(), request, depth+1)}

	case *types.Map:
		return &Schema{Type: "object", Values: s.of(u.Elem(), request, depth+1)}

	case *types.Struct:
		named, ok := t.(*types.Named)
		if !ok {
			return &Schema{Type: "object", Properties: s.fields(u, request, depth)}
		}

		key := structKey{named.Obj(), request}
		if schema, ok := s.structs[key]; ok {
			return schema
		}

		schema := &Schema{Type: "object", Name: s.name(named.Obj()), Description: s.loader.typeDoc(named.Obj())}
		s.structs[key] = schema
		s.named[schema.Name] = schema
		schema.Properties = s.fields(u, request, depth)
		return schema
	}

	return &Schema{}
}

func (s *schemas) fields(st *types.Struct, request bool, depth int) []Property {
	var out []Property
	wire := map[string]string{}

	for i := range st.NumFields() {
		field := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))

		name, options, _ := strings.Cut(tag.Get("json"), ",")
		if name == "-" {
			continue
		}

		if field.Embedded() && name == "" {
			if inner, ok := field.Type().Underlying().(*types.Struct); ok {
				out = append(out, s.fields(inner, request, depth)...)
				continue
			}
		}

		if !field.Exported() {
			continue
		}
		if name == "" {
			name = field.Name()
		}

		schema := s.of(field.Type(), request, depth+1)
		rules, required := validation(tag.Get("validate"), schema)
		if !request {
			required = !strings.Contains(options, "omitempty") && !schema.Nullable
		}

		wire[field.Name()] = name
		out = append(out, Property{
			Name:        name,
			Schema:      schema,
			Required:    required,
			Rules:       rules,
			Description: s.loader.fieldDoc(field),
		})
	}

	rename := wireNames(wire)
	for i, property := range out {
		out[i].Description = rename(property.Description)
		for j, rule := range property.Rules {
			out[i].Rules[j] = rename(rule)
		}
	}

	return out
}

func (s *schemas) name(obj *types.TypeName) string {
	if name, ok := s.seen[obj]; ok {
		return name
	}

	name := exported(obj.Name())
	for _, taken := range s.seen {
		if taken == name {
			name = exported(obj.Pkg().Name()) + name
			break
		}
	}
	s.seen[obj] = name

	return name
}

func (s *schemas) Components() []*Schema {
	out := make([]*Schema, 0, len(s.named))
	for _, schema := range s.named {
		out = append(out, schema)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func basic(b *types.Basic) *Schema {
	info := b.Info()
	switch {
	case info&types.IsBoolean != 0:
		return &Schema{Type: "boolean"}
	case info&types.IsInteger != 0:
		return &Schema{Type: "integer"}
	case info&types.IsFloat != 0:
		return &Schema{Type: "number"}
	case info&types.IsString != 0:
		return &Schema{Type: "string"}
	}
	return &Schema{}
}

func wellKnown(t *types.Named) *Schema {
	if t.Obj().Pkg() == nil {
		return nil
	}

	switch t.Obj().Pkg().Path() + "." + t.Obj().Name() {
	case "time.Time":
		return &Schema{Type: "string", Format: "date-time"}
	case "time.Duration":
		return &Schema{Type: "integer", Description: "nanoseconds"}
	case "github.com/google/uuid.UUID":
		return &Schema{Type: "string", Format: "uuid"}
	case "encoding/json.Number":
		return &Schema{Type: "number"}
	case "encoding/json.RawMessage", "gorm.io/datatypes.JSON":
		return &Schema{}
	case "database/sql.NullString":
		return &Schema{Type: "string", Nullable: true}
	case "database/sql.NullTime", "gorm.io/gorm.DeletedAt":
		return &Schema{Type: "string", Format: "date-time", Nullable: true}
	}

	return nil
}

func validation(tag string, schema *Schema) (rules []string, required bool) {
	if tag == "" {
		return nil, false
	}

	unit := "characters"
	switch schema.Type {
	case "array":
		unit = "items"
	case "integer", "number":
		unit = ""
	}

	for rule := range strings.SplitSeq(tag, ",") {
		name, value, _ := strings.Cut(rule, "=")
		switch name {
		case "required":
			required = true
		case "email":
			rules = append(rules, "an email address")
		case "url", "http_url":
			rules = append(rules, "a URL")
		case "uuid", "uuid4", "uuid7":
			rules = append(rules, "a UUID")
		case "max", "lte":
			rules = append(rules, strings.TrimSpace("at most "+value+" "+unit))
		case "min", "gte":
			rules = append(rules, strings.TrimSpace("at least "+value+" "+unit))
		case "len":
			rules = append(rules, strings.TrimSpace("exactly "+value+" "+unit))
		case "oneof":
			schema.Enum = strings.Fields(value)
		case "eqfield":
			rules = append(rules, "the same as "+value)
		case "nefield":
			rules = append(rules, "not the same as "+value)
		case "required_with", "required_if", "required_unless", "required_without":
			rules = append(rules, strings.ReplaceAll(name, "_", " ")+" "+value)
		case "dive", "omitempty":
		default:
			if value != "" {
				rules = append(rules, name+" "+value)
			} else {
				rules = append(rules, name)
			}
		}
	}

	return rules, required
}

func wireNames(wire map[string]string) func(string) string {
	names := make([]string, 0, len(wire))
	for goName := range wire {
		names = append(names, regexp.QuoteMeta(goName))
	}
	if len(names) == 0 {
		return func(text string) string { return text }
	}
	// Longest first: the pattern takes the first alternative that matches,
	// so IsPasswordTemporary has to be tried before Password.
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	pattern := regexp.MustCompile(`\b(` + strings.Join(names, "|") + `)\b`)

	return func(text string) string {
		return pattern.ReplaceAllStringFunc(text, func(goName string) string {
			return "`" + wire[goName] + "`"
		})
	}
}

func (l *loader) enum(t *types.Named) []string {
	pkg := t.Obj().Pkg()
	if pkg == nil || l.packages[pkg.Path()] == nil {
		return nil
	}

	var values []string
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if !ok || !types.Identical(c.Type(), t) || c.Val().Kind() != constant.String {
			continue
		}
		values = append(values, constant.StringVal(c.Val()))
	}
	sort.Strings(values)

	return values
}

func (l *loader) fieldDoc(field *types.Var) string {
	node := l.fields[field.Pos()]
	if node == nil {
		return ""
	}
	if node.Doc != nil {
		return sentence(node.Doc.Text())
	}
	if node.Comment != nil {
		return sentence(node.Comment.Text())
	}
	return ""
}

func (l *loader) typeDoc(obj *types.TypeName) string {
	if doc := l.types[obj.Pos()]; doc != nil {
		return sentence(doc.Text())
	}
	return ""
}

func (l *loader) indexDeclarations(file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.GenDecl:
			if n.Tok != token.TYPE {
				return true
			}
			for _, spec := range n.Specs {
				ts := spec.(*ast.TypeSpec)
				doc := ts.Doc
				if doc == nil && len(n.Specs) == 1 {
					doc = n.Doc
				}
				l.types[ts.Name.Pos()] = doc
			}
		case *ast.Field:
			for _, name := range n.Names {
				l.fields[name.Pos()] = n
			}
			if len(n.Names) == 0 {
				l.fields[n.Type.Pos()] = n
			}
		}
		return true
	})
}

func sentence(text string) string {
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")
	for i, paragraph := range paragraphs {
		if strings.HasPrefix(paragraph, "\t") || strings.HasPrefix(paragraph, "  ") ||
			strings.Contains(paragraph, "\n\t") || strings.Contains(paragraph, "\n  -") {
			paragraphs[i] = paragraph
			continue
		}
		paragraphs[i] = strings.Join(strings.Fields(paragraph), " ")
	}
	return strings.Join(paragraphs, "\n\n")
}

func exported(name string) string {
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func statusOf(info *types.Info, expr ast.Expr) int {
	tv, ok := info.Types[expr]
	if !ok || tv.Value == nil {
		return 0
	}
	n, err := strconv.Atoi(tv.Value.ExactString())
	if err != nil {
		return 0
	}
	return n
}

func stringOf(info *types.Info, expr ast.Expr) (string, bool) {
	tv, ok := info.Types[expr]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(tv.Value), true
}
