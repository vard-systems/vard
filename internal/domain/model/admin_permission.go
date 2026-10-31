package model

// AdminPermission is one thing an administrator may do. The catalog is code,
// not a table, because each permission is a guarded route; a name that leaves
// the catalog stops granting anything.
type AdminPermission struct {
	Name        string `json:"name"`
	Group       string `json:"group"`
	Description string `json:"description"`

	// Scopable permissions can be granted for one application; a scoped role
	// grants only these.
	Scopable bool `json:"scopable"`
}

const (
	PermActivityRead    = "activity.read"
	PermUsersRead       = "users.read"
	PermUsersWrite      = "users.write"
	PermUserFieldsWrite = "user_fields.write"

	PermAPIsRead  = "apis.read"
	PermAPIsWrite = "apis.write"

	PermOrganizationRead  = "organization.read"
	PermOrganizationWrite = "organization.write"

	PermSocialRead  = "social.read"
	PermSocialWrite = "social.write"

	PermSSORead  = "sso.read"
	PermSSOWrite = "sso.write"

	PermLoginFlowsRead  = "login_flows.read"
	PermLoginFlowsWrite = "login_flows.write"

	PermLanguagesRead  = "languages.read"
	PermLanguagesWrite = "languages.write"

	PermApplicationsRead     = "applications.read"
	PermApplicationsWrite    = "applications.write"
	PermUserRolesWrite       = "user_roles.write"
	PermRoleAssignmentsWrite = "role_assignments.write"
)

var AdminPermissions = []AdminPermission{
	{
		Name:        PermActivityRead,
		Group:       "Activity",
		Description: "See the dashboard counts and the activity log",
	},
	{
		Name:        PermUsersRead,
		Group:       "Users",
		Description: "See users and their fields",
	},
	{
		Name:        PermUsersWrite,
		Group:       "Users",
		Description: "Create, edit and delete users",
	},
	{
		Name:        PermUserFieldsWrite,
		Group:       "Users",
		Description: "Add, change and remove user fields",
	},
	{
		Name:        PermOrganizationRead,
		Group:       "Organization",
		Description: "See the organisation this installation belongs to",
	},
	{
		Name:        PermOrganizationWrite,
		Group:       "Organization",
		Description: "Change the organisation's name, domain, support address and logo",
	},
	{
		Name:        PermSocialRead,
		Group:       "Authentication",
		Description: "See the providers users can sign in with",
	},
	{
		Name:  PermSocialWrite,
		Group: "Authentication",
		Description: "Register providers users can sign in with, and change their keys: " +
			"whoever holds this decides which accounts elsewhere reach this server",
	},
	{
		Name:        PermLoginFlowsRead,
		Group:       "Authentication",
		Description: "See the login flows applications sign their users in with",
	},
	{
		Name:  PermLoginFlowsWrite,
		Group: "Authentication",
		Description: "Write login flows and decide which is the default: " +
			"whoever holds this decides what a sign-in asks for",
	},
	{
		Name:        PermSSORead,
		Group:       "Authentication",
		Description: "See the organisations' identity providers, their domains and how they map roles",
	},
	{
		Name:  PermSSOWrite,
		Group: "Authentication",
		Description: "Connect, change and remove identity providers: " +
			"whoever holds this decides who may sign in, as whom, and with which roles",
	},
	{
		Name:        PermLanguagesRead,
		Group:       "Languages",
		Description: "See the languages and read their text",
	},
	{
		Name:        PermLanguagesWrite,
		Group:       "Languages",
		Description: "Add, translate and remove languages, and decide which are offered and the default",
	},
	{
		Name:        PermAPIsRead,
		Group:       "APIs",
		Description: "See APIs, their scopes, and which applications may use them",
	},
	{
		Name:        PermAPIsWrite,
		Group:       "APIs",
		Description: "Register, change and remove APIs and their scopes",
	},
	{
		Name:        PermApplicationsRead,
		Group:       "Applications",
		Description: "See applications and the roles they define",
		Scopable:    true,
	},
	{
		Name:  PermApplicationsWrite,
		Group: "Applications",
		Description: "Change an application's settings and rotate its secret; " +
			"granted for every application, also create and delete applications",
		Scopable: true,
	},
	{
		Name:  PermUserRolesWrite,
		Group: "Roles",
		Description: "Create, edit and delete roles: an application's when held for it, " +
			"and the global roles too when held for the whole panel",
		Scopable: true,
	},
	{
		Name:  PermRoleAssignmentsWrite,
		Group: "Roles",
		Description: "Give users roles and take them away: an application's when held for it, " +
			"and the global roles too when held for the whole panel",
		Scopable: true,
	},
}

func IsAdminPermission(name string) bool {
	_, ok := adminPermission(name)
	return ok
}

func IsScopablePermission(name string) bool {
	permission, ok := adminPermission(name)
	return ok && permission.Scopable
}

func adminPermission(name string) (AdminPermission, bool) {
	for _, permission := range AdminPermissions {
		if permission.Name == name {
			return permission, true
		}
	}

	return AdminPermission{}, false
}

func AdminPermissionNames() []string {
	names := make([]string, 0, len(AdminPermissions))
	for _, permission := range AdminPermissions {
		names = append(names, permission.Name)
	}

	return names
}
