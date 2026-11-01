package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFullName(t *testing.T) {
	for _, tt := range []struct {
		first, last, want string
	}{
		{"Ada", "Lovelace", "Ada Lovelace"},
		{"Ada", "", "Ada"},
		{"", "Lovelace", "Lovelace"},
		{"", "", ""},
	} {
		if got := (Admin{FirstName: tt.first, LastName: tt.last}).FullName(); got != tt.want {
			t.Errorf("FullName(%q, %q) = %q, want %q", tt.first, tt.last, got, tt.want)
		}
	}
}

func TestCanSignIn(t *testing.T) {
	tests := []struct {
		name  string
		admin Admin
		want  bool
	}{
		{
			name:  "active account",
			admin: Admin{Status: StatusActive},
			want:  true,
		},
		{
			name:  "invited account has not set a password yet",
			admin: Admin{Status: StatusInvited},
		},
		{
			name:  "suspended account",
			admin: Admin{Status: StatusSuspended},
		},
		{
			name:  "disabled account",
			admin: Admin{Status: StatusDisabled},
		},
		{
			name:  "active account",
			admin: Admin{Status: StatusActive},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.admin.CanSignIn(); got != tt.want {
				t.Errorf("CanSignIn() = %v, want %v", got, tt.want)
			}
		})
	}
}

// global assigns a role for the whole panel.
func global(name string, permissions ...string) AdminRoleAssignment {
	return AdminRoleAssignment{Role: AdminRole{Name: name, Permissions: permissions}}
}

// scoped assigns a role for one application.
func scoped(app uuid.UUID, name string, permissions ...string) AdminRoleAssignment {
	return AdminRoleAssignment{Role: AdminRole{Name: name, Permissions: permissions}, ApplicationID: &app}
}

func TestHasRole(t *testing.T) {
	shop := uuid.New()
	admin := Admin{Assignments: []AdminRoleAssignment{
		global("support"),
		scoped(shop, RoleSuperAdmin),
	}}

	if !admin.HasRole("support") {
		t.Error("HasRole(support) = false, want true")
	}
	if admin.HasRole("moderator") {
		t.Error("HasRole(moderator) = true, want false")
	}

	// super_admin only counts for the whole panel: a scoped one, however it
	// got stored, does not make someone a super admin.
	if admin.IsSuperAdmin() {
		t.Error("IsSuperAdmin() = true for a super_admin scoped to one app, want false")
	}

	if (Admin{}).HasRole("moderator") {
		t.Error("an admin with no assignments loaded reported having a role")
	}
}

func TestHasPermission(t *testing.T) {
	shop, blog := uuid.New(), uuid.New()

	tests := []struct {
		name       string
		admin      Admin
		permission string
		app        *uuid.UUID // nil asks about the whole panel
		want       bool
	}{
		{
			name:       "a global role grants it",
			admin:      Admin{Assignments: []AdminRoleAssignment{global("support", PermUsersRead)}},
			permission: PermUsersRead,
			want:       true,
		},
		{
			name:       "a global role does not grant it",
			admin:      Admin{Assignments: []AdminRoleAssignment{global("support", PermUsersRead)}},
			permission: PermUsersWrite,
		},
		{
			name: "the second of two roles grants it",
			admin: Admin{Assignments: []AdminRoleAssignment{
				global("auditor", PermActivityRead),
				global("support", PermUsersWrite),
			}},
			permission: PermUsersWrite,
			want:       true,
		},
		{
			name:       "a global role grants it for any one application",
			admin:      Admin{Assignments: []AdminRoleAssignment{global("admin", PermUserRolesWrite)}},
			permission: PermUserRolesWrite,
			app:        &shop,
			want:       true,
		},
		{
			name:       "a role scoped to the application grants it there",
			admin:      Admin{Assignments: []AdminRoleAssignment{scoped(shop, "owner", PermUserRolesWrite)}},
			permission: PermUserRolesWrite,
			app:        &shop,
			want:       true,
		},
		{
			name:       "a role scoped to another application does not",
			admin:      Admin{Assignments: []AdminRoleAssignment{scoped(shop, "owner", PermUserRolesWrite)}},
			permission: PermUserRolesWrite,
			app:        &blog,
		},
		{
			name:       "a scoped role does not grant it for the whole panel",
			admin:      Admin{Assignments: []AdminRoleAssignment{scoped(shop, "owner", PermUserRolesWrite)}},
			permission: PermUserRolesWrite,
		},
		{
			name:       "a scoped role never grants a permission that cannot be scoped",
			admin:      Admin{Assignments: []AdminRoleAssignment{scoped(shop, "owner", PermUsersWrite)}},
			permission: PermUsersWrite,
			app:        &shop,
		},
		{
			name:       "super admin may do anything, with no permissions listed",
			admin:      Admin{Assignments: []AdminRoleAssignment{global(RoleSuperAdmin)}},
			permission: "anything.at.all",
			want:       true,
		},
		{
			name:       "no roles at all",
			admin:      Admin{},
			permission: PermUsersRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.admin.HasPermissionFor(tt.permission, tt.app); got != tt.want {
				t.Errorf("HasPermissionFor(%q) = %v, want %v", tt.permission, got, tt.want)
			}
		})
	}
}

// ApplicationsWith is what a list narrows itself by: everything for a global
// grant, and only the scoped applications otherwise.
func TestApplicationsWith(t *testing.T) {
	shop, blog := uuid.New(), uuid.New()

	owner := Admin{Assignments: []AdminRoleAssignment{
		scoped(shop, "owner", PermApplicationsRead),
		scoped(blog, "support", PermUsersRead),
	}}

	all, ids := owner.ApplicationsWith(PermApplicationsRead)
	if all || len(ids) != 1 || ids[0] != shop {
		t.Errorf("ApplicationsWith = %v, %v; want shop only", all, ids)
	}
	if !owner.HasPermissionAnywhere(PermApplicationsRead) || owner.HasPermissionAnywhere(PermUserRolesWrite) {
		t.Error("HasPermissionAnywhere does not follow the scoped grants")
	}

	scopedPerms := owner.ScopedPermissions()
	if len(scopedPerms) != 1 || len(scopedPerms[shop]) != 1 || scopedPerms[shop][0] != PermApplicationsRead {
		t.Errorf("ScopedPermissions() = %v, want applications.read for shop only", scopedPerms)
	}

	super := Admin{Assignments: []AdminRoleAssignment{global(RoleSuperAdmin)}}
	if all, _ := super.ApplicationsWith(PermApplicationsRead); !all {
		t.Error("a super admin should reach every application")
	}
}

// Permissions is what the panel is drawn from, so it lists catalog names
// only, in catalog order, and a super admin gets every one of them.
func TestPermissions(t *testing.T) {
	moderator := Admin{Assignments: []AdminRoleAssignment{
		global("moderator", PermUsersWrite, "retired.permission", PermActivityRead),
		scoped(uuid.New(), "owner", PermUserRolesWrite),
	}}

	got := moderator.Permissions()
	want := []string{PermActivityRead, PermUsersWrite}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Permissions() = %v, want %v", got, want)
	}

	super := Admin{Assignments: []AdminRoleAssignment{global(RoleSuperAdmin)}}
	if got := super.Permissions(); len(got) != len(AdminPermissions) {
		t.Errorf("super admin Permissions() = %v, want the whole catalog", got)
	}
}

func TestAdminPermissionCatalog(t *testing.T) {
	seen := map[string]bool{}
	for _, permission := range AdminPermissions {
		if permission.Name == "" || permission.Group == "" || permission.Description == "" {
			t.Errorf("%+v is missing a name, group or description", permission)
		}
		if seen[permission.Name] {
			t.Errorf("%s is listed twice", permission.Name)
		}
		seen[permission.Name] = true

		if !IsAdminPermission(permission.Name) {
			t.Errorf("IsAdminPermission(%q) = false", permission.Name)
		}
	}

	if IsAdminPermission("admins.manage") {
		t.Error("managing admins is super_admin's alone and must not be grantable")
	}

	// Users are shared by every application, so what is done to them cannot
	// be limited to one.
	if IsScopablePermission(PermUsersWrite) || !IsScopablePermission(PermUserRolesWrite) {
		t.Error("the scopable permissions are not the application ones")
	}
}

func TestSessionIsActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name    string
		session AdminSession
		want    bool
	}{
		{
			name:    "current session that cleared MFA",
			session: AdminSession{ExpiresAt: future, IsMFAPassed: true},
			want:    true,
		},
		{
			name:    "expired",
			session: AdminSession{ExpiresAt: past, IsMFAPassed: true},
		},
		{
			name:    "revoked",
			session: AdminSession{ExpiresAt: future, IsMFAPassed: true, RevokedAt: &past},
		},
		{
			name:    "half-finished login that never cleared MFA",
			session: AdminSession{ExpiresAt: future},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsActive(now); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMFAIsConfirmed(t *testing.T) {
	now := time.Now()

	if (MFA{}).IsConfirmed() {
		t.Error("an unconfirmed factor reported itself confirmed")
	}
	if !(MFA{ConfirmedAt: &now}).IsConfirmed() {
		t.Error("a confirmed factor reported itself unconfirmed")
	}
}

// Software calling the admin API with a token is seen as an administrator
// holding exactly its token's permissions, for the whole panel, and never a
// super admin — so the routes' guards judge it like anyone else.
func TestServiceAdmin(t *testing.T) {
	app := Application{Name: "admin-cli", ClientID: AdminCLIClientID}
	admin := ServiceAdmin(app, []string{PermUsersRead})

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"it holds its token's permission", admin.HasPermission(PermUsersRead), true},
		{"and no other", admin.HasPermission(PermUsersWrite), false},
		{"for every application", admin.HasPermissionAnywhere(PermUsersRead), true},
		{"it is never a super admin", admin.IsSuperAdmin(), false},
		{"it is a service", admin.Service, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}

	if admin.Username != AdminCLIClientID {
		t.Errorf("Username = %q, want the client id the log names", admin.Username)
	}
}
