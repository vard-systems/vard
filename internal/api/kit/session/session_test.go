package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/model"
)

// serve runs one request through a guard, as the administrator given (or
// nobody), and returns the status it was answered with.
func serve(guard gin.HandlerFunc, admin *model.Admin) int {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		if admin != nil {
			c.Set(adminKey, admin)
		}
		c.Next()
	}, guard, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	return w.Code
}

// assigned is an administrator holding these roles for the whole panel.
func assigned(roles ...model.AdminRole) *model.Admin {
	admin := &model.Admin{}
	for _, role := range roles {
		admin.Assignments = append(admin.Assignments, model.AdminRoleAssignment{Role: role})
	}
	return admin
}

func TestCan(t *testing.T) {
	support := assigned(model.AdminRole{Name: "support", Permissions: []string{model.PermUsersRead}})
	super := assigned(model.AdminRole{Name: model.RoleSuperAdmin})

	tests := []struct {
		name       string
		admin      *model.Admin
		permission string
		want       int
	}{
		{name: "a role grants it", admin: support, permission: model.PermUsersRead, want: http.StatusNoContent},
		{name: "no role grants it", admin: support, permission: model.PermUsersWrite, want: http.StatusForbidden},
		{name: "a super admin", admin: super, permission: model.PermUsersWrite, want: http.StatusNoContent},
		{name: "nobody signed in", permission: model.PermUsersRead, want: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serve(Can(tt.permission), tt.admin); got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

// Every permission in the catalog is not enough to manage administrators:
// that takes the super_admin role itself.
func TestRequireSuperAdmin(t *testing.T) {
	everything := assigned(model.AdminRole{Name: "admin", Permissions: model.AdminPermissionNames()})
	super := assigned(model.AdminRole{Name: model.RoleSuperAdmin})

	if got := serve(RequireSuperAdmin(), everything); got != http.StatusForbidden {
		t.Errorf("admin with every permission: status = %d, want 403", got)
	}
	if got := serve(RequireSuperAdmin(), super); got != http.StatusNoContent {
		t.Errorf("super admin: status = %d, want 204", got)
	}
	if got := serve(RequireSuperAdmin(), nil); got != http.StatusForbidden {
		t.Errorf("nobody: status = %d, want 403", got)
	}
}

// scopedTo is an administrator holding a role for one application only.
func scopedTo(app uuid.UUID, role model.AdminRole) *model.Admin {
	return &model.Admin{Assignments: []model.AdminRoleAssignment{{Role: role, ApplicationID: &app}}}
}

// as is a request context carrying the administrator, as Require leaves it.
func as(admin *model.Admin) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if admin != nil {
		c.Set(adminKey, admin)
	}

	return c
}

func TestCanAnywhere(t *testing.T) {
	shop := uuid.New()
	manager := scopedTo(shop, model.AdminRole{Name: "app_manager", Permissions: []string{model.PermApplicationsRead}})

	if got := serve(CanAnywhere(model.PermUsersRead, model.PermApplicationsRead), manager); got != http.StatusNoContent {
		t.Errorf("one of the permissions for one application: status = %d, want 204", got)
	}
	if got := serve(CanAnywhere(model.PermUsersRead), manager); got != http.StatusForbidden {
		t.Errorf("none of the permissions: status = %d, want 403", got)
	}

	// A permission that cannot be scoped grants nothing when held for one
	// application.
	usersForShop := scopedTo(shop, model.AdminRole{Name: "support", Permissions: []string{model.PermUsersRead}})
	if got := serve(CanAnywhere(model.PermUsersRead), usersForShop); got != http.StatusForbidden {
		t.Errorf("an unscopable permission held for one application: status = %d, want 403", got)
	}
}

func TestScopedChecks(t *testing.T) {
	shop, blog := uuid.New(), uuid.New()
	manager := as(scopedTo(shop, model.AdminRole{
		Name:        "app_manager",
		Permissions: []string{model.PermApplicationsRead, model.PermUserRolesWrite},
	}))

	if !Allowed(manager, model.PermUserRolesWrite, shop) {
		t.Error("Allowed for their own application = false")
	}
	if Allowed(manager, model.PermUserRolesWrite, blog) {
		t.Error("Allowed for another application = true")
	}

	if !AllowedScope(manager, model.PermUserRolesWrite, &shop) {
		t.Error("AllowedScope for their application's roles = false")
	}
	if AllowedScope(manager, model.PermUserRolesWrite, nil) {
		t.Error("AllowedScope for the global roles = true, want it to take the whole panel")
	}

	if reach := Reach(manager, model.PermApplicationsRead); len(reach) != 1 || reach[0] != shop {
		t.Errorf("Reach = %v, want only their application", reach)
	}
	if reach := Reach(as(assigned(model.AdminRole{Name: model.RoleSuperAdmin})), model.PermApplicationsRead); reach != nil {
		t.Errorf("Reach for a super admin = %v, want nil for every application", reach)
	}
	if reach := Reach(as(nil), model.PermApplicationsRead); reach == nil || len(reach) != 0 {
		t.Errorf("Reach for nobody = %v, want an empty list", reach)
	}

	global := model.UserRole{Name: "employee"}
	shopRole := model.UserRole{Name: "viewer", ApplicationID: &shop}
	blogRole := model.UserRole{Name: "writer", ApplicationID: &blog}

	if !SeesRole(manager, global) || !SeesRole(manager, shopRole) || SeesRole(manager, blogRole) {
		t.Error("SeesRole: want the global roles and their application's, and not another's")
	}
}

func TestToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{name: "no cookie", header: "", want: ""},
		{name: "one cookie", header: Cookie + "=abc", want: "abc"},
		{name: "another cookie beside it", header: "theme=dark; " + Cookie + "=abc", want: "abc"},
		{name: "a second one planted from a sibling subdomain", header: Cookie + "=planted; " + Cookie + "=abc", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				c.Request.Header.Set("Cookie", tc.header)
			}

			if got := Token(c); got != tc.want {
				t.Errorf("Token = %q, want %q", got, tc.want)
			}
		})
	}
}
