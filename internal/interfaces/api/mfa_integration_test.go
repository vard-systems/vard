package api

import (
	"net/http"
	"testing"
	"time"

	"vard/internal/config"
	"vard/internal/domain/totp"
)

// codeFor is the authenticator's code for a moment.
func codeFor(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := totp.Code(secret, totp.Step(at))
	if err != nil {
		t.Fatal(err)
	}
	return code
}

type loginAnswer struct {
	Next  string         `json:"next"`
	Admin map[string]any `json:"admin"`
}

func (c *client) loginAnswer(email, password string) (int, loginAnswer) {
	var out loginAnswer
	status := c.do(http.MethodPost, "/auth/login", map[string]string{"username": email, "password": password}, &out)
	return status, out
}

// mfaStatus is what an administrator is told about their own second factor.
type mfaStatus struct {
	MFA struct {
		Enabled           bool   `json:"enabled"`
		Required          bool   `json:"required"`
		RecoveryCodesLeft int    `json:"recovery_codes_left"`
		LastUsedAt        string `json:"last_used_at"`
	} `json:"mfa"`
}

func (c *client) state() string {
	var out struct {
		State string `json:"state"`
	}
	c.must(http.StatusOK, http.MethodGet, "/auth/session", nil, &out)
	return out.State
}

// Where two-factor sign-in is required, an administrator without it is made to
// set it up before anything else, and signs in with it from then on.
func TestLiveAdminMFARequired(t *testing.T) {
	s := newLiveServerWith(t, func(cfg *config.Config) { cfg.AdminMFARequired = true })

	root := s.client()
	root.must(http.StatusCreated, http.MethodPost, "/setup", map[string]string{
		"email": superEmail, "password": superPassword, "first_name": "Root",
		"setup_token": s.setupToken,
	}, nil)

	// The password alone only gets as far as setting up.
	status, answer := root.loginAnswer(superEmail, superPassword)
	if status != http.StatusOK || answer.Next != "enroll" || answer.Admin != nil {
		t.Fatalf("login = %d %+v, want next=enroll and no admin", status, answer)
	}
	if got := root.state(); got != "enroll" {
		t.Errorf("state = %q, want enroll", got)
	}
	for _, path := range []string{"/me", "/users", "/admins"} {
		if status := root.do(http.MethodGet, path, nil, nil); status != http.StatusUnauthorized {
			t.Errorf("GET %s before setting up = %d, want 401", path, status)
		}
	}

	var begun struct {
		Enrolment struct {
			Secret string `json:"secret"`
			URI    string `json:"uri"`
		} `json:"enrolment"`
	}
	root.must(http.StatusOK, http.MethodPost, "/mfa/totp", nil, &begun)
	secret := begun.Enrolment.Secret
	if secret == "" || begun.Enrolment.URI == "" {
		t.Fatalf("enrolment = %+v", begun)
	}

	if status := root.do(http.MethodPost, "/mfa/totp/confirm", map[string]string{"code": "000000"}, nil); status != http.StatusBadRequest {
		t.Errorf("confirming with a wrong code = %d, want 400", status)
	}

	var confirmed struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	now := time.Now()
	root.must(http.StatusOK, http.MethodPost, "/mfa/totp/confirm", map[string]string{"code": codeFor(t, secret, now)}, &confirmed)
	if len(confirmed.RecoveryCodes) != 10 {
		t.Fatalf("recovery codes = %v, want 10", confirmed.RecoveryCodes)
	}

	// Confirming signed the waiting session in.
	var me struct {
		Admin struct {
			MFAEnabled bool `json:"mfa_enabled"`
		} `json:"admin"`
	}
	root.must(http.StatusOK, http.MethodGet, "/me", nil, &me)
	if !me.Admin.MFAEnabled {
		t.Error("me.mfa_enabled = false after confirming")
	}

	// It cannot be turned off where it is required.
	if status := root.do(http.MethodDelete, "/mfa/totp", map[string]string{"code": confirmed.RecoveryCodes[9]}, nil); status != http.StatusConflict {
		t.Errorf("turning off a required factor = %d, want 409", status)
	}

	// Next sign-in: a code is needed, and the used step cannot be replayed.
	laptop := s.client()
	if _, answer := laptop.loginAnswer(superEmail, superPassword); answer.Next != "mfa" {
		t.Fatalf("second login next = %q, want mfa", answer.Next)
	}
	if status := laptop.do(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("me waiting for a code = %d, want 401", status)
	}
	// A session that has the password but not the phone cannot enrol its own
	// authenticator.
	if status := laptop.do(http.MethodPost, "/mfa/totp", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("starting a set-up while waiting for a code = %d, want 401", status)
	}

	if status := laptop.do(http.MethodPost, "/auth/mfa", map[string]string{"code": codeFor(t, secret, now)}, nil); status != http.StatusUnauthorized {
		t.Errorf("replaying the code used to confirm = %d, want 401", status)
	}

	next := codeFor(t, secret, now.Add(totp.Period))
	laptop.must(http.StatusOK, http.MethodPost, "/auth/mfa", map[string]string{"code": next}, nil)
	laptop.must(http.StatusOK, http.MethodGet, "/me", nil, nil)

	// What the authenticator last spent, read before a recovery code is used.
	var before mfaStatus
	laptop.must(http.StatusOK, http.MethodGet, "/mfa", nil, &before)
	if before.MFA.LastUsedAt == "" {
		t.Fatal("the factor says it has never been used, after two codes from it")
	}

	// A recovery code works once.
	phone := s.client()
	phone.loginAnswer(superEmail, superPassword)
	phone.must(http.StatusOK, http.MethodPost, "/auth/mfa", map[string]string{"code": confirmed.RecoveryCodes[0]}, nil)

	again := s.client()
	again.loginAnswer(superEmail, superPassword)
	if status := again.do(http.MethodPost, "/auth/mfa", map[string]string{"code": confirmed.RecoveryCodes[0]}, nil); status != http.StatusUnauthorized {
		t.Errorf("reusing a recovery code = %d, want 401", status)
	}

	var status2 mfaStatus
	laptop.must(http.StatusOK, http.MethodGet, "/mfa", nil, &status2)
	if !status2.MFA.Enabled || !status2.MFA.Required || status2.MFA.RecoveryCodesLeft != 9 {
		t.Errorf("mfa status = %+v", status2.MFA)
	}

	// A recovery code is not a step of the authenticator, and does not spend
	// one. Writing the moment it was used into that column would make the step
	// it fell in look replayed, and the code showing on the administrator's
	// phone would be refused until the next one appeared.
	if status2.MFA.LastUsedAt != before.MFA.LastUsedAt {
		t.Errorf("the last step used went from %q to %q over a recovery code, want it left alone",
			before.MFA.LastUsedAt, status2.MFA.LastUsedAt)
	}

	// A super admin resets another administrator's factor: they are signed out
	// and set up again at their next sign-in.
	const staffEmail, staffPassword = "staff@example.com", "staff-password-1"
	laptop.must(http.StatusCreated, http.MethodPost, "/admins", map[string]any{
		"email": staffEmail, "first_name": "Staff", "status": "active",
		"password": staffPassword, "confirm_password": staffPassword,
		"assignments": []map[string]any{{"role_id": laptop.adminRoleID("auditor")}},
	}, nil)

	staff := s.client()
	staff.loginAnswer(staffEmail, staffPassword)
	var staffBegun struct {
		Enrolment struct {
			Secret string `json:"secret"`
		} `json:"enrolment"`
	}
	staff.must(http.StatusOK, http.MethodPost, "/mfa/totp", nil, &staffBegun)
	staff.must(http.StatusOK, http.MethodPost, "/mfa/totp/confirm", map[string]string{"code": codeFor(t, staffBegun.Enrolment.Secret, time.Now())}, nil)
	staff.must(http.StatusOK, http.MethodGet, "/me", nil, nil)

	var admins struct {
		Admins []struct {
			ID         string `json:"id"`
			Email      string `json:"email"`
			MFAEnabled bool   `json:"mfa_enabled"`
		} `json:"admins"`
	}
	laptop.must(http.StatusOK, http.MethodGet, "/admins", nil, &admins)
	staffID := ""
	for _, a := range admins.Admins {
		if a.Email == staffEmail {
			staffID = a.ID
			if !a.MFAEnabled {
				t.Error("the admin list says staff has no second factor")
			}
		}
	}

	var self struct {
		Admin struct {
			ID string `json:"id"`
		} `json:"admin"`
	}
	laptop.must(http.StatusOK, http.MethodGet, "/me", nil, &self)
	if status := laptop.do(http.MethodDelete, "/admins/"+self.Admin.ID+"/mfa", nil, nil); status != http.StatusConflict {
		t.Errorf("resetting your own factor = %d, want 409", status)
	}

	laptop.must(http.StatusOK, http.MethodDelete, "/admins/"+staffID+"/mfa", nil, nil)
	if status := staff.do(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("staff after a reset = %d, want signed out", status)
	}
	if _, answer := staff.loginAnswer(staffEmail, staffPassword); answer.Next != "enroll" {
		t.Errorf("staff login after a reset next = %q, want enroll", answer.Next)
	}
}

// Where it is optional, an administrator signs in with the password alone
// until they turn it on, and can turn it off again with a code.
func TestLiveAdminMFAOptional(t *testing.T) {
	s := newLiveServer(t)
	root := s.superAdmin()

	// An installation set up with VARD_ADMIN_MFA=optional asks nobody for a
	// second factor: the first administrator is signed in by their password
	// alone. (A fresh installation without it requires one.)
	if got := root.state(); got != "signed_in" {
		t.Fatalf("the first sign-in is at %q, want signed_in", got)
	}

	var policy struct {
		MFARequired bool `json:"require_mfa"`
	}
	root.must(http.StatusOK, http.MethodGet, "/security", nil, &policy)
	if policy.MFARequired {
		t.Error("an installation set up optional requires a second factor")
	}

	// And an administrator who wants one sets it up themselves.
	var begun struct {
		Enrolment struct {
			Secret string `json:"secret"`
		} `json:"enrolment"`
	}
	root.must(http.StatusOK, http.MethodPost, "/mfa/totp", nil, &begun)
	now := time.Now()
	root.must(http.StatusOK, http.MethodPost, "/mfa/totp/confirm", map[string]string{"code": codeFor(t, begun.Enrolment.Secret, now)}, nil)

	// Replacing it takes a code from the current one.
	if status := root.do(http.MethodPost, "/mfa/totp", nil, nil); status != http.StatusConflict {
		t.Errorf("starting a replacement without a code = %d, want 409", status)
	}

	var codes struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	root.must(http.StatusOK, http.MethodPost, "/mfa/recovery-codes", map[string]string{"code": codeFor(t, begun.Enrolment.Secret, now.Add(totp.Period))}, &codes)
	if len(codes.RecoveryCodes) != 10 {
		t.Fatalf("regenerated codes = %v", codes.RecoveryCodes)
	}

	root.must(http.StatusNoContent, http.MethodDelete, "/mfa/totp", map[string]string{"code": codes.RecoveryCodes[0]}, nil)

	other := s.client()
	if _, answer := other.loginAnswer(superEmail, superPassword); answer.Admin == nil || answer.Next != "" {
		t.Errorf("login after turning it off = %+v, want signed in with the password", answer)
	}
}

// Requiring a second factor is a setting a super admin changes, not a line in
// a file: turning it on makes the administrators who have none set one up
// before they can do anything else, and turning it off lets them remove it.
func TestLiveAdminMFAPolicyIsManaged(t *testing.T) {
	s := newLiveServerWith(t, func(cfg *config.Config) { cfg.AdminMFARequired = false })
	super := s.superAdmin()

	type securityBody struct {
		MFARequired    bool  `json:"require_mfa"`
		Administrators int64 `json:"administrators"`
		WithMFA        int64 `json:"with_mfa"`
	}

	// What the installation started with, and what it would mean to change.
	var current securityBody
	super.must(http.StatusOK, http.MethodGet, "/security", nil, &current)

	if current.MFARequired {
		t.Error("the setting did not start as the configuration said")
	}
	if current.Administrators != 1 || current.WithMFA != 0 {
		t.Errorf("counts = %+v, want the one administrator, with no authenticator", current)
	}

	// Turned on, the administrator who has none is let no further than
	// setting one up — on the session they already hold.
	var updated securityBody
	super.must(http.StatusOK, http.MethodPatch, "/security", map[string]any{"require_mfa": true}, &updated)
	if !updated.MFARequired {
		t.Fatal("the setting did not change")
	}

	var state struct {
		State       string `json:"state"`
		MFARequired bool   `json:"mfa_required"`
	}
	super.must(http.StatusOK, http.MethodGet, "/auth/session", nil, &state)

	if state.State != "enroll" || !state.MFARequired {
		t.Errorf("the session is %+v, want one that has to set an authenticator up", state)
	}

	// And nothing else is open to it while that is so.
	super.must(http.StatusUnauthorized, http.MethodGet, "/admins", nil, nil)

	// A sign-in from a fresh browser is held at the same place.
	second := s.client()
	if status := second.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("signing in = %d", status)
	}
	second.must(http.StatusOK, http.MethodGet, "/auth/session", nil, &state)
	if state.State != "enroll" {
		t.Errorf("a new sign-in is at %q, want enroll", state.State)
	}

	// Not even to undo it: a session that has to set an authenticator up can
	// do nothing else, the setting that made it so included. The way out is
	// to set one up, which is what the panel warns about before turning it on.
	super.must(http.StatusUnauthorized, http.MethodPatch, "/security", map[string]any{"require_mfa": false}, nil)

	// So: set one up, and then it can be turned off again.
	var begun struct {
		Enrolment struct {
			Secret string `json:"secret"`
		} `json:"enrolment"`
	}
	super.must(http.StatusOK, http.MethodPost, "/mfa/totp", nil, &begun)
	super.must(http.StatusOK, http.MethodPost, "/mfa/totp/confirm", map[string]string{
		"code": codeFor(t, begun.Enrolment.Secret, time.Now()),
	}, nil)

	var off securityBody
	super.must(http.StatusOK, http.MethodPatch, "/security", map[string]any{"require_mfa": false}, &off)

	if off.MFARequired || off.WithMFA != 1 {
		t.Errorf("settings = %+v, want it off and the one authenticator counted", off)
	}

	// With it off, an administrator may remove their authenticator again —
	// which is what the requirement was stopping. The code is the next one:
	// the one that confirmed the authenticator cannot be used twice.
	super.must(http.StatusNoContent, http.MethodDelete, "/mfa/totp", map[string]string{
		// One whole period on, which is always the next step. A second more
		// than one — which this said — is two steps on whenever the clock is
		// in the last second of a step, and two is past the skew the server
		// allows, so the code was refused about once in thirty runs.
		"code": codeFor(t, begun.Enrolment.Secret, time.Now().Add(totp.Period)),
	}, nil)
}
