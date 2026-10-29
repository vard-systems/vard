package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/google/uuid"

	"vard/internal/brand"
	"vard/internal/cache"
	"vard/internal/cache/cachetest"
	"vard/internal/model"
)

func TestLiveWrongPasswordsMakeAnAdministratorWait(t *testing.T) {
	s := newLiveServer(t)
	s.superAdmin()

	c := s.client()
	for range 5 {
		if status := c.login(superEmail, "wrong-password"); status != http.StatusUnauthorized {
			t.Fatalf("wrong password = %d, want 401", status)
		}
	}

	if status := c.login(superEmail, superPassword); status != http.StatusUnauthorized {
		t.Errorf("right password on a waiting account = %d, want 401", status)
	}
}

func TestLiveLoginResetsFailures(t *testing.T) {
	s := newLiveServer(t)
	s.superAdmin()

	c := s.client()
	for range 4 {
		c.login(superEmail, "wrong-password")
	}
	if status := c.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("login after four failures = %d, want 200", status)
	}

	// The count started again, so four more are not a wait either.
	for range 4 {
		c.login(superEmail, "wrong-password")
	}
	if status := c.login(superEmail, superPassword); status != http.StatusOK {
		t.Errorf("login after four more failures = %d, want 200", status)
	}
}

// An administrator made to wait shows on the overview, and a password a
// super admin sets is the way back in straight away.
func TestLiveASuperAdminsPasswordEndsTheWait(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	const email, password = "waiting@example.com", "waiting-password-1"
	var created struct {
		Admin idOnly `json:"admin"`
	}
	body := map[string]any{
		"email": email, "first_name": "Wait", "status": "active",
		"password": password, "confirm_password": password,
		"assignments": []map[string]any{},
	}
	super.must(http.StatusCreated, http.MethodPost, "/admins", body, &created)

	for range model.ThrottleFree {
		s.client().login(email, "wrong-password")
	}
	if status := s.client().login(email, password); status != http.StatusUnauthorized {
		t.Fatalf("the right password while waiting = %d, want 401", status)
	}

	var got overviewBody
	super.must(http.StatusOK, http.MethodGet, "/overview", nil, &got)
	if got.Counts["locked_admins"] != 1 {
		t.Errorf("the overview counts %d administrators waiting, want 1", got.Counts["locked_admins"])
	}

	const fresh = "a-fresh-password-1"
	body["password"], body["confirm_password"] = fresh, fresh
	super.must(http.StatusOK, http.MethodPatch, "/admins/"+created.Admin.ID, body, nil)

	if status := s.client().login(email, fresh); status != http.StatusOK {
		t.Errorf("the password a super admin set = %d, want 200 at once", status)
	}
}

// An administrator changes their own account from the profile: a name
// freely, a new address and a new password only with the one they have. A
// new password signs every other browser out and keeps the one it came from.
func TestLiveOwnAccount(t *testing.T) {
	s := newLiveServer(t)
	here := s.superAdmin()

	elsewhere := s.client()
	if status := elsewhere.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("second sign-in = %d", status)
	}

	var answer struct {
		Admin struct {
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			FullName  string `json:"full_name"`
		} `json:"admin"`
	}
	here.must(http.StatusOK, http.MethodPatch, "/me", map[string]string{
		"first_name": "Ada", "last_name": "Lovelace", "email": superEmail,
	}, &answer)
	if answer.Admin.FullName != "Ada Lovelace" || answer.Admin.LastName != "Lovelace" {
		t.Errorf("after renaming = %+v, want Ada Lovelace", answer.Admin)
	}

	const moved = "ada@example.com"
	for _, tt := range []struct {
		name     string
		password string
		want     int
	}{
		{"a new address without the password", "", http.StatusBadRequest},
		{"a new address with the wrong password", "not-the-password", http.StatusBadRequest},
		{"a new address with the password", superPassword, http.StatusOK},
	} {
		status := here.do(http.MethodPatch, "/me", map[string]string{
			"first_name": "Ada", "email": moved, "current_password": tt.password,
		}, nil)
		if status != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, status, tt.want)
		}
	}

	if status := s.client().login(moved, superPassword); status != http.StatusOK {
		t.Errorf("sign-in with the new address = %d, want 200", status)
	}

	const next = "a-brand-new-password"
	if status := here.do(http.MethodPost, "/me/password", map[string]string{
		"current_password": "not-the-password", "new_password": next,
	}, nil); status != http.StatusBadRequest {
		t.Errorf("new password with the wrong current one = %d, want 400", status)
	}
	if status := here.do(http.MethodPost, "/me/password", map[string]string{
		"current_password": superPassword, "new_password": "short",
	}, nil); status != http.StatusBadRequest {
		t.Errorf("a new password that is too short = %d, want 400", status)
	}
	here.must(http.StatusOK, http.MethodPost, "/me/password", map[string]string{
		"current_password": superPassword, "new_password": next,
	}, nil)

	if status := here.do(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Errorf("the session that changed the password = %d, want still signed in", status)
	}
	if status := elsewhere.do(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("another session after a new password = %d, want 401", status)
	}
	if status := s.client().login(moved, next); status != http.StatusOK {
		t.Errorf("sign-in with the new password = %d, want 200", status)
	}
}

// An administrator of one application manages its roles, but cannot reach
// the global roles or another application's through them.
func TestLiveScopedAdminStaysInTheirApplication(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	shop := super.application("shop")
	blog := super.application("blog")
	employee := super.role("employee", "")
	blogWriter := super.role("writer", blog)

	manager := s.appManager(super, shop)

	// Their own application's roles are theirs to make.
	viewer := manager.role("viewer", shop)
	manager.role("editor", shop, viewer)

	// Including a global role would hand it to everyone holding the role.
	status := manager.do(http.MethodPost, "/user-roles", map[string]any{
		"name": "staff", "application_id": shop, "inherits": []string{employee},
	}, nil)
	if status != http.StatusForbidden {
		t.Errorf("app role including a global role = %d, want 403", status)
	}

	// Another application's roles are not even there.
	status = manager.do(http.MethodPost, "/user-roles", map[string]any{
		"name": "staff", "application_id": shop, "inherits": []string{blogWriter},
	}, nil)
	if status != http.StatusBadRequest {
		t.Errorf("app role including another app's role = %d, want 400", status)
	}

	if status := manager.do(http.MethodPost, "/user-roles", map[string]any{"name": "intruder", "application_id": blog}, nil); status != http.StatusBadRequest {
		t.Errorf("role in another application = %d, want 400", status)
	}

	if status := manager.do(http.MethodPost, "/user-roles", map[string]any{"name": "intruder"}, nil); status != http.StatusForbidden {
		t.Errorf("global role = %d, want 403", status)
	}

	if status := manager.do(http.MethodGet, "/applications/"+blog, nil, nil); status != http.StatusNotFound {
		t.Errorf("another application = %d, want 404", status)
	}

	if status := manager.do(http.MethodGet, "/admins", nil, nil); status != http.StatusForbidden {
		t.Errorf("administrators = %d, want 403", status)
	}

	// A super admin may still compose across scopes.
	super.role("staff", shop, employee)
}

// An inclusion someone else made stays when a scoped administrator edits the
// rest of the role.
func TestLiveScopedAdminKeepsExistingInclusions(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	shop := super.application("shop")
	employee := super.role("employee", "")
	staff := super.role("staff", shop, employee)

	manager := s.appManager(super, shop)

	manager.must(http.StatusOK, http.MethodPatch, "/user-roles/"+staff, map[string]any{
		"name": "staff", "description": "Everyone at the shop", "inherits": []string{employee},
	}, nil)
}

type apiScope struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     bool   `json:"is_default"`
}

type apiBody struct {
	ID     string     `json:"id"`
	Scopes []apiScope `json:"scopes"`
}

func TestLiveAPIScopesCanTradeNames(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	var created struct {
		API apiBody `json:"api"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/apis", map[string]any{
		"name": "Orders", "identifier": "https://api.example.com/orders",
		"scopes": []apiScope{{Name: "orders:read"}, {Name: "orders:write"}},
	}, &created)

	read, write := created.API.Scopes[0], created.API.Scopes[1]
	read.Name, write.Name = write.Name, read.Name

	var updated struct {
		API apiBody `json:"api"`
	}
	super.must(http.StatusOK, http.MethodPatch, "/apis/"+created.API.ID, map[string]any{
		"name": "Orders", "scopes": []apiScope{read, write},
	}, &updated)

	names := map[string]string{}
	for _, scope := range updated.API.Scopes {
		names[scope.ID] = scope.Name
	}
	if names[read.ID] != "orders:write" || names[write.ID] != "orders:read" {
		t.Errorf("scopes after trading names = %v, want the ids kept and the names swapped", names)
	}

	// A new scope may take the name of one removed in the same save: read is
	// called orders:write by now.
	super.must(http.StatusOK, http.MethodPatch, "/apis/"+created.API.ID, map[string]any{
		"name": "Orders", "scopes": []apiScope{write, {Name: "orders:write"}},
	}, nil)
}

// Deleting an application takes its roles with it, the API scopes they grant
// included.
func TestLiveDeleteApplicationWithRoleGrants(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	var api struct {
		API apiBody `json:"api"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/apis", map[string]any{
		"name": "Orders", "identifier": "https://api.example.com/orders",
		"scopes": []apiScope{{Name: "orders:read"}},
	}, &api)

	shop := super.application("shop")
	super.must(http.StatusCreated, http.MethodPost, "/user-roles", map[string]any{
		"name": "buyer", "application_id": shop, "api_scopes": []string{api.API.Scopes[0].ID},
	}, nil)
	super.must(http.StatusOK, http.MethodPut, "/applications/"+shop+"/apis/"+api.API.ID, map[string]any{
		"scopes": []string{api.API.Scopes[0].ID},
	}, nil)

	super.must(http.StatusNoContent, http.MethodDelete, "/applications/"+shop, nil, nil)

	var roles struct {
		Total int `json:"total"`
	}
	super.must(http.StatusOK, http.MethodGet, "/user-roles", nil, &roles)
	if roles.Total != 0 {
		t.Errorf("roles after deleting their application = %d, want 0", roles.Total)
	}

	super.must(http.StatusNoContent, http.MethodDelete, "/apis/"+api.API.ID, nil, nil)
}

func TestLiveSearchTreatsWildcardsLiterally(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	super.role("first_line", "")
	super.role("firstaline", "")

	var out struct {
		Total int `json:"total"`
	}
	super.must(http.StatusOK, http.MethodGet, "/user-roles?search="+url.QueryEscape("first_"), nil, &out)
	if out.Total != 1 {
		t.Errorf("search for first_ = %d roles, want only first_line", out.Total)
	}
}

type overviewBody struct {
	Counts  map[string]int64 `json:"counts"`
	SignIns struct {
		Succeeded int64 `json:"succeeded"`
		Failed    int64 `json:"failed"`
		Blocked   int64 `json:"blocked"`
	} `json:"sign_ins"`
	Daily []struct {
		Day      string `json:"day"`
		Events   int64  `json:"events"`
		Failures int64  `json:"failures"`
	} `json:"daily"`
	TopActors []struct {
		Actor  string `json:"actor"`
		Events int64  `json:"events"`
	} `json:"top_actors"`
	Activity []struct {
		Action string `json:"action"`
		Detail string `json:"detail"`
		Target *struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"target"`
	} `json:"activity"`
}

func TestLiveOverview(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	shop := super.application("shop")
	super.role("viewer", shop)
	s.client().login(superEmail, "wrong-password")
	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "user@example.com", "password": "user-password-1", "confirm_password": "user-password-1",
		"is_email_verified": true,
	}, nil)
	user := s.browser()
	if status := user.account(http.MethodPost, "/login", map[string]string{
		"email": "user@example.com", "password": "wrong-password",
	}, nil); status != http.StatusUnauthorized {
		t.Fatalf("user login with a wrong password = %d, want 401", status)
	}
	if status := user.account(http.MethodPost, "/login", map[string]string{
		"email": "user@example.com", "password": "user-password-1",
	}, nil); status != http.StatusOK {
		t.Fatalf("user login = %d, want 200", status)
	}

	var got overviewBody
	super.must(http.StatusOK, http.MethodGet, "/overview", nil, &got)

	// Two applications: the test's, and admin-cli, which every installation
	// starts with.
	if got.Counts["applications"] != 2 || got.Counts["enabled_applications"] != 2 || got.Counts["user_roles"] != 1 || got.Counts["admins"] != 1 {
		t.Errorf("counts = %v", got.Counts)
	}
	if got.Counts["active_sessions"] != 1 {
		t.Errorf("active_sessions = %d, want the super admin's one", got.Counts["active_sessions"])
	}

	if got.SignIns.Succeeded != 2 || got.SignIns.Failed != 2 || got.SignIns.Blocked != 0 {
		t.Errorf("sign-ins = %+v, want two successes, two failures and no blocked attempts", got.SignIns)
	}

	if len(got.Daily) != 14 {
		t.Fatalf("daily = %d days, want 14", len(got.Daily))
	}
	if today := got.Daily[13]; today.Events < 7 || today.Failures != 2 {
		t.Errorf("today = %+v, want user and admin events plus two failures", today)
	}

	if len(got.TopActors) != 1 || got.TopActors[0].Actor != superEmail || got.TopActors[0].Events != 3 {
		t.Errorf("top actors = %+v, want the super admin with three changes and no sign-ins", got.TopActors)
	}

	named := map[string]string{}
	for _, event := range got.Activity {
		if event.Target != nil {
			named[event.Action] = event.Target.Name
		}
		if event.Action == "admin.login_failed" && event.Detail != "wrong password" {
			t.Errorf("failed sign-in detail = %q, want the reason", event.Detail)
		}
	}
	if named["application.created"] != "shop" || named["user_role.created"] != "viewer" {
		t.Errorf("targets named = %v, want the application and the role", named)
	}
}

// An administrator who may read the log but not a kind of record sees that
// something happened to it, without what it is called.
func TestLiveActivityHidesNamesAdministratorsCannotSee(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	super.application("shop")

	const email, password = "auditor@example.com", "auditor-password-1"
	var created struct {
		Role idOnly `json:"role"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/admin-roles", map[string]any{
		"name": "log_reader", "permissions": []string{"activity.read"},
	}, &created)
	super.must(http.StatusCreated, http.MethodPost, "/admins", map[string]any{
		"email": email, "first_name": "Log", "status": "active",
		"password": password, "confirm_password": password,
		"assignments": []map[string]any{{"role_id": created.Role.ID}},
	}, nil)

	reader := s.client()
	if status := reader.login(email, password); status != http.StatusOK {
		t.Fatalf("login = %d", status)
	}

	var got overviewBody
	reader.must(http.StatusOK, http.MethodGet, "/overview", nil, &got)

	for _, event := range got.Activity {
		if event.Target != nil && event.Target.Name != "" {
			t.Errorf("%s: target named %q, want no name for this administrator", event.Action, event.Target.Name)
		}
	}
}

// A social provider is named in the log, like an SSO connection, for whoever
// may read social providers, and for nobody else.
func TestLiveActivityNamesSocialProviders(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	super.must(http.StatusCreated, http.MethodPost, "/social-providers", map[string]any{
		"kind": "google", "slug": "google", "name": "Google Workspace", "client_id": "id", "client_secret": "secret",
	}, nil)

	const email, password = "auditor@example.com", "auditor-password-1"
	var created struct {
		Role idOnly `json:"role"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/admin-roles", map[string]any{
		"name": "log_reader", "permissions": []string{"activity.read"},
	}, &created)
	super.must(http.StatusCreated, http.MethodPost, "/admins", map[string]any{
		"email": email, "first_name": "Log", "status": "active",
		"password": password, "confirm_password": password,
		"assignments": []map[string]any{{"role_id": created.Role.ID}},
	}, nil)
	reader := s.client()
	if status := reader.login(email, password); status != http.StatusOK {
		t.Fatalf("login = %d", status)
	}

	tests := []struct {
		name string
		who  *client
		want string
	}{
		{"a super admin", super, "Google Workspace"},
		{"an administrator who cannot read social providers", reader, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got overviewBody
			tt.who.must(http.StatusOK, http.MethodGet, "/overview", nil, &got)

			for _, event := range got.Activity {
				if event.Action == "social_provider.created" {
					if event.Target == nil || event.Target.Name != tt.want {
						t.Errorf("target = %+v, want named %q", event.Target, tt.want)
					}
					return
				}
			}
			t.Error("the provider's creation is not in the activity")
		})
	}
}

// A panel with nothing done this week still answers with lists, not null,
// which the dashboard counts the length of.
func TestLiveOverviewListsAreNeverNull(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	var raw map[string]json.RawMessage
	super.must(http.StatusOK, http.MethodGet, "/overview", nil, &raw)

	for _, key := range []string{"daily", "top_actors", "activity"} {
		if value := string(raw[key]); value == "null" || value == "" || value[0] != '[' {
			t.Errorf("%s = %s, want a list", key, value)
		}
	}
}

// The organisation is one record of settings every installation starts with:
// an update changes what it names and leaves the rest as it was, and the log
// says which settings moved.
// A stale organisation in the cache — cached before the database was reset,
// or by another server sharing the Redis — is not what a save writes back:
// the save reads the row itself, and succeeds.
func TestLiveOrganizationSaveIgnoresAStaleCache(t *testing.T) {
	s := newLiveServer(t)
	if s.cache == nil {
		t.Skip("no test Redis: set " + cachetest.Env)
	}
	super := s.superAdmin()

	// Something reads the organisation first, so the group is in use.
	super.must(http.StatusOK, http.MethodGet, "/organization", nil, nil)

	stale := model.DefaultOrganization()
	stale.ID = uuid.New()
	stale.Name = "stale"
	s.cache.Cache.Set(context.Background(), cache.Organization, "settings", stale)

	var answer struct {
		Organization struct {
			Name     string `json:"name"`
			TermsURL string `json:"terms_url"`
		} `json:"organization"`
	}
	super.must(http.StatusOK, http.MethodPatch, "/organization", map[string]any{
		"terms_url": "https://acme.example.com/terms",
	}, &answer)

	if answer.Organization.TermsURL != "https://acme.example.com/terms" || answer.Organization.Name != brand.Name {
		t.Errorf("after the save = %+v, want the stored organization with its terms", answer.Organization)
	}

	stored, err := s.store.Organization(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stored.TermsURL != "https://acme.example.com/terms" {
		t.Errorf("stored terms = %q, want the saved link", stored.TermsURL)
	}
}

// Two administrators saving different settings at the same moment both keep
// what they saved: neither save writes back a row the other has overtaken.
func TestLiveOrganizationSavesAtOnceKeepEachOthersFields(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	for round := range 10 {
		email := fmt.Sprintf("help%d@acme.example.com", round)
		phone := fmt.Sprintf("+1 555 01%02d", round)

		var wg sync.WaitGroup
		statuses := make([]int, 2)
		for i, change := range []map[string]any{{"support_email": email}, {"support_phone": phone}} {
			wg.Go(func() {
				statuses[i] = super.do(http.MethodPatch, "/organization", change, nil)
			})
		}
		wg.Wait()

		if statuses[0] != http.StatusOK || statuses[1] != http.StatusOK {
			t.Fatalf("round %d: saves = %v, want both 200", round, statuses)
		}

		var answer struct {
			Organization struct {
				SupportEmail string `json:"support_email"`
				SupportPhone string `json:"support_phone"`
			} `json:"organization"`
		}
		super.must(http.StatusOK, http.MethodGet, "/organization", nil, &answer)

		if answer.Organization.SupportEmail != email || answer.Organization.SupportPhone != phone {
			t.Fatalf("round %d: stored %+v, want %q and %q", round, answer.Organization, email, phone)
		}
	}
}

// The one-time code settings are read from the cache everywhere else; a save
// changes the stored row, so a stale cached copy cannot undo another setting.
func TestLiveOTPSaveIgnoresAStaleCache(t *testing.T) {
	s := newLiveServer(t)
	if s.cache == nil {
		t.Skip("no test Redis: set " + cachetest.Env)
	}
	super := s.superAdmin()

	super.must(http.StatusOK, http.MethodPatch, "/otp", map[string]any{"max_attempts": 3}, nil)

	stale := model.DefaultOTPSettings()
	stale.ID = uuid.New()
	s.cache.Cache.Set(context.Background(), cache.OTPSettings, "settings", stale)

	super.must(http.StatusOK, http.MethodPatch, "/otp", map[string]any{"resend_seconds": 90}, nil)

	stored, err := s.store.OTPSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stored.MaxAttempts != 3 || stored.ResendSeconds != 90 {
		t.Errorf("stored = %d attempts, %d seconds; want 3 and 90", stored.MaxAttempts, stored.ResendSeconds)
	}
}

func TestLiveOrganizationSettings(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	type organizationBody struct {
		Organization struct {
			Name         string `json:"name"`
			Slug         string `json:"slug"`
			Timezone     string `json:"timezone"`
			SupportEmail string `json:"support_email"`
			SupportPhone string `json:"support_phone"`
			TermsURL     string `json:"terms_url"`
			PrivacyURL   string `json:"privacy_url"`
		} `json:"organization"`
	}

	// The migration seeds the row, so the page has something to open on.
	var seeded organizationBody
	super.must(http.StatusOK, http.MethodGet, "/organization", nil, &seeded)

	if seeded.Organization.Name != brand.Name || seeded.Organization.Slug != brand.Slug || seeded.Organization.Timezone != "UTC" {
		t.Errorf("seeded = %+v, want the default organization", seeded.Organization)
	}

	var updated organizationBody
	super.must(http.StatusOK, http.MethodPatch, "/organization", map[string]any{
		"name":          "  Acme Inc  ",
		"slug":          "Acme",
		"timezone":      "Asia/Bishkek",
		"support_email": "support@acme.example.com",
		"support_phone": "+996 555 123456",
		"terms_url":     "https://acme.example.com/terms",
		"privacy_url":   "https://acme.example.com/privacy",
	}, &updated)

	if got := updated.Organization; got.Name != "Acme Inc" || got.Slug != "acme" || got.Timezone != "Asia/Bishkek" {
		t.Errorf("updated = %+v, want the values trimmed and lower cased", got)
	}

	// The panel's header reads the name from the session's own answer, which
	// every administrator can ask for.
	var me struct {
		Organization struct {
			Name string `json:"name"`
		} `json:"organization"`
	}
	super.must(http.StatusOK, http.MethodGet, "/me", nil, &me)
	if me.Organization.Name != "Acme Inc" {
		t.Errorf("/me organization = %q, want the saved name", me.Organization.Name)
	}

	// A second update says nothing about the timezone, which therefore stays.
	var kept organizationBody
	super.must(http.StatusOK, http.MethodPatch, "/organization", map[string]any{
		"support_phone": "+996 555 999888",
	}, &kept)

	if kept.Organization.Timezone != "Asia/Bishkek" || kept.Organization.SupportPhone != "+996 555 999888" {
		t.Errorf("kept = %+v, want the timezone left alone and the new number", kept.Organization)
	}

	// A refused change writes nothing.
	super.must(http.StatusBadRequest, http.MethodPatch, "/organization", map[string]any{
		"slug": "acme inc",
	}, nil)

	var reread organizationBody
	super.must(http.StatusOK, http.MethodGet, "/organization", nil, &reread)
	if reread.Organization.Slug != "acme" {
		t.Errorf("slug = %q, want the stored one after a refused change", reread.Organization.Slug)
	}

	// The log says which settings moved, newest first.
	var logs struct {
		Logs []struct {
			Action string `json:"action"`
			Target *struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			} `json:"target"`
			Detail string `json:"detail"`
		} `json:"logs"`
	}
	super.must(http.StatusOK, http.MethodGet, "/logs?limit=50", nil, &logs)

	var changed []string
	for _, entry := range logs.Logs {
		if entry.Action == "organization.updated" {
			if entry.Target == nil || entry.Target.Type != "organization" || entry.Target.ID != "acme" {
				t.Errorf("target = %+v, want the organization by its slug", entry.Target)
			}
			changed = append(changed, entry.Detail)
		}
	}
	if len(changed) != 2 {
		t.Fatalf("organization.updated entries = %d, want one per change that took", len(changed))
	}
	if changed[0] != "support_phone" {
		t.Errorf("the last change recorded %q, want the number alone", changed[0])
	}

	// A role scoped to one application grants nothing here: these are the
	// whole installation's settings.
	manager := s.appManager(super, super.application("shop"))
	manager.must(http.StatusForbidden, http.MethodGet, "/organization", nil, nil)
	manager.must(http.StatusForbidden, http.MethodPatch, "/organization", map[string]any{"name": "Theirs"}, nil)

	// What an administrator saved is what the outside world is told. The
	// agreements are the provider's own metadata, and the sign-in pages read
	// the rest from the public API, which takes no session.
	discovery := map[string]any{}
	getJSON(t, s.root+"/.well-known/openid-configuration", &discovery)

	if discovery["op_tos_uri"] != "https://acme.example.com/terms" {
		t.Errorf("op_tos_uri = %v, want the organization's terms", discovery["op_tos_uri"])
	}
	if discovery["op_policy_uri"] != "https://acme.example.com/privacy" {
		t.Errorf("op_policy_uri = %v, want the organization's privacy policy", discovery["op_policy_uri"])
	}

	var public struct {
		Organization struct {
			Name         string `json:"name"`
			SupportEmail string `json:"support_email"`
			SupportPhone string `json:"support_phone"`
			TermsURL     string `json:"terms_url"`
		} `json:"organization"`
	}
	getJSON(t, s.root+"/api/v1/account/organization", &public)

	if got := public.Organization; got.Name != "Acme Inc" || got.SupportEmail != "support@acme.example.com" {
		t.Errorf("the sign-in pages are told %+v, want the saved name and address", got)
	}
	if public.Organization.SupportPhone != "+996 555 999888" {
		t.Errorf("support_phone = %q, want the number saved last", public.Organization.SupportPhone)
	}
}

// getJSON reads a public endpoint, which needs no session and no cookie.
func getJSON(t *testing.T, url string, out any) {
	t.Helper()

	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", url, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		t.Fatalf("GET %s: decode: %v", url, err)
	}
}

// Ready is a database that answers; a server whose database has gone is
// taken out of service with a defined problem, not left to fail each request.
func TestLiveReadyFollowsTheDatabase(t *testing.T) {
	s := newLiveServer(t)

	if status := raw(t, http.MethodGet, s.root+"/readyz", "", nil); status != http.StatusOK {
		t.Fatalf("readyz with the database up = %d, want 200", status)
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	if status := raw(t, http.MethodGet, s.root+"/readyz", "", nil); status != http.StatusServiceUnavailable {
		t.Errorf("readyz with the database gone = %d, want 503", status)
	}
}
