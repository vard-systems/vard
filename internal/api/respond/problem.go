package respond

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"loginer/i18n"
)

type Problem struct {
	Status int
	Code   string
	Apps   []i18n.App
}

var (
	Public = []i18n.App{i18n.ID}
	Admin  = []i18n.App{i18n.Console}
	Both   = []i18n.App{i18n.ID, i18n.Console}
)

var (
	registryMu sync.Mutex
	registry   = map[string]Problem{}
)

// Define registers a problem from a package-level var beside its handler. A
// duplicate code stops the server at startup.
func Define(status int, code string, apps []i18n.App) Problem {
	registryMu.Lock()
	defer registryMu.Unlock()

	if _, taken := registry[code]; taken {
		panic(fmt.Sprintf("respond: the problem %q is defined twice", code))
	}

	problem := Problem{Status: status, Code: code, Apps: apps}
	registry[code] = problem

	return problem
}

func Lookup(code string) (Problem, bool) {
	registryMu.Lock()
	defer registryMu.Unlock()

	problem, ok := registry[code]

	return problem, ok
}

func Problems() []Problem {
	registryMu.Lock()
	defer registryMu.Unlock()

	out := make([]Problem, 0, len(registry))
	for _, problem := range registry {
		out = append(out, problem)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })

	return out
}

func (p Problem) Key() string {
	return "error." + p.Code
}

func (p Problem) English(params map[string]any) string {
	for _, app := range p.Apps {
		if text, ok := i18n.Text(app, p.Key()); ok {
			return i18n.Fill(text, params)
		}
	}

	return p.Code
}

func (p Problem) With(pairs ...any) Fault {
	params := map[string]any{}
	for i := 0; i+1 < len(pairs); i += 2 {
		params[fmt.Sprint(pairs[i])] = pairs[i+1]
	}

	return p.Fault(params)
}

func (p Problem) Fault(params map[string]any) Fault {
	if len(params) == 0 {
		params = nil
	}

	return Fault{Status: p.Status, Code: p.Code, Params: params, Message: p.English(params)}
}

var (
	InvalidBody = Define(http.StatusBadRequest, "invalid_body", Both)
	NotFoundAny = Define(http.StatusNotFound, "not_found", Both)
	Internal    = Define(http.StatusInternalServerError, "internal", Both)
	NotSignedIn = Define(http.StatusUnauthorized, "not_signed_in", Both)
	NotAllowed  = Define(http.StatusForbidden, "forbidden", Both)

	LanguageNotFound = Define(http.StatusNotFound, "language_not_found", Both)
)
