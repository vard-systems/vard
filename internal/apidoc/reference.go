// Package apidoc writes the API reference from the code that serves the API:
// the route table, each handler's doc comment, request types and the problems
// defined beside them. TestTheReferenceIsCurrent fails when the committed
// reference is behind.
//
// It writes an OpenAPI 3.1 document per server, which the server answers.
// `make docs` runs it.
package apidoc

type Reference struct {
	Servers  []Server
	Problems []Problem
}

type Server struct {
	Key      string
	Title    string
	Sections []Section
}

func (s Server) Operations() []Operation {
	var out []Operation
	for _, section := range s.Sections {
		out = append(out, section.Operations...)
	}
	return out
}

type Section struct {
	Slug       string
	Title      string
	Intro      string
	Operations []Operation
}

type Operation struct {
	Method        string
	Path          string
	Handler       string
	Summary       string
	Description   string
	Access        Access
	RateLimit     string
	OriginChecked bool

	PathParams  []Param
	QueryParams []Param
	FormParams  []Param
	Headers     []Param
	BasicAuth   bool

	Body      *Schema
	Responses []Response
	Problems  []Problem
}

type Access struct {
	Session     string
	Token       string
	Permissions []string
	Anywhere    bool
	SuperAdmin  bool
}

type Param struct {
	Name        string
	Description string
	Required    bool
	Schema      *Schema
}

type Response struct {
	Status      int
	ContentType string
	Schema      *Schema
}

type Problem struct {
	Status  int
	Code    string
	English string
	Apps    []string
}
