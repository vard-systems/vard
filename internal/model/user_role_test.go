package model

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// graph builds a role graph from names: each role inherits the roles listed
// for it.
func graph(t *testing.T, roles map[string][]string) (RoleGraph, map[string]uuid.UUID) {
	t.Helper()

	ids := map[string]uuid.UUID{}
	for name := range roles {
		ids[name] = uuid.New()
	}

	g := RoleGraph{}
	for name, inherits := range roles {
		role := UserRole{Name: name,
			ID: ids[name]}

		for _, inherited := range inherits {
			parent := UserRole{Name: inherited,
				ID: ids[inherited]}
			role.Inherits = append(role.Inherits, parent)
		}

		g[role.ID] = role
	}

	return g, ids
}

func TestRoleGraphEffective(t *testing.T) {
	g, ids := graph(t, map[string][]string{
		"viewer": nil,
		"editor": {"viewer"},
		"admin":  {"editor"},
		"other":  nil,
	})

	got := Names(g.Effective([]uuid.UUID{ids["admin"], uuid.New()}))

	if want := []string{"admin", "editor", "viewer"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Effective(admin) = %v, want %v", got, want)
	}
}

// A cycle stored by some other route must not hang resolving it.
func TestRoleGraphEffectiveSurvivesACycle(t *testing.T) {
	g, ids := graph(t, map[string][]string{
		"a": {"b"},
		"b": {"a"},
	})

	if got, want := Names(g.Effective([]uuid.UUID{ids["a"]})), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Effective(a) = %v, want %v", got, want)
	}
}

func TestRoleGraphWouldCycle(t *testing.T) {
	g, ids := graph(t, map[string][]string{
		"viewer": nil,
		"editor": {"viewer"},
		"admin":  {"editor"},
		"other":  nil,
	})

	tests := []struct {
		name     string
		role     string
		inherits []string
		want     bool
	}{
		{name: "itself", role: "viewer", inherits: []string{"viewer"}, want: true},
		{name: "its own child", role: "viewer", inherits: []string{"editor"}, want: true},
		{name: "a grandchild", role: "viewer", inherits: []string{"admin"}, want: true},
		{name: "an unrelated role", role: "viewer", inherits: []string{"other"}},
		{name: "what it already inherits", role: "admin", inherits: []string{"editor", "viewer"}},
		{name: "nothing", role: "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inherits []uuid.UUID
			for _, name := range tt.inherits {
				inherits = append(inherits, ids[name])
			}

			if got := g.WouldCycle(ids[tt.role], inherits); got != tt.want {
				t.Errorf("WouldCycle(%s, %v) = %v, want %v", tt.role, tt.inherits, got, tt.want)
			}
		})
	}
}

func TestRoleNamePattern(t *testing.T) {
	for _, name := range []string{"viewer", "support-tier_2", "a"} {
		if !RoleNamePattern.MatchString(name) {
			t.Errorf("role name %q refused, want it allowed", name)
		}
	}
	for _, name := range []string{"", "Viewer", "2nd", "has space", "posts:read"} {
		if RoleNamePattern.MatchString(name) {
			t.Errorf("role name %q allowed, want it refused", name)
		}
	}
}

func TestUserRoleScope(t *testing.T) {
	shop, blog := uuid.New(), uuid.New()

	global := UserRole{Name: "employee"}
	shopAdmin := UserRole{Name: "admin", ApplicationID: &shop}
	shopViewer := UserRole{Name: "viewer", ApplicationID: &shop}
	blogAdmin := UserRole{Name: "admin", ApplicationID: &blog}

	if !global.Global() || shopAdmin.Global() {
		t.Error("Global() does not follow the application")
	}
	if !shopAdmin.In(&shop) || shopAdmin.In(&blog) || shopAdmin.In(nil) || !global.In(nil) || global.In(&shop) {
		t.Error("In() does not follow the application")
	}

	tests := []struct {
		name        string
		role, other UserRole
		want        bool
	}{
		{"a global role includes a global role", global, UserRole{Name: "staff"}, true},
		{"a global role includes an application role", global, shopAdmin, true},
		{"an application role includes a global role", shopAdmin, global, true},
		{"an application role includes its own application's", shopAdmin, shopViewer, true},
		{"an application role includes another application's", shopAdmin, blogAdmin, false},
	}

	for _, tt := range tests {
		if got := tt.role.MayInherit(tt.other); got != tt.want {
			t.Errorf("%s: MayInherit = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// Via is what the role mapping tab says next to an inherited role: the roles
// the user was given that it comes through.
func TestRoleGraphVia(t *testing.T) {
	g, ids := graph(t, map[string][]string{
		"viewer": nil,
		"editor": {"viewer"},
		"admin":  {"editor"},
		"other":  nil,
	})

	via := g.Via([]uuid.UUID{ids["admin"], ids["viewer"], ids["other"]})

	if got := via[ids["viewer"]]; len(got) != 1 || got[0] != ids["admin"] {
		t.Errorf("viewer via %v, want admin: it is also reached through admin", got)
	}
	if got := via[ids["editor"]]; len(got) != 1 || got[0] != ids["admin"] {
		t.Errorf("editor via %v, want admin", got)
	}
	if got := via[ids["other"]]; len(got) != 0 {
		t.Errorf("other via %v, want nothing: it was only given directly", got)
	}
	if got := via[ids["admin"]]; len(got) != 0 {
		t.Errorf("admin via %v, want nothing", got)
	}
}
