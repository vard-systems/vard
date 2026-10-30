package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
)

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()

	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}

	return out.Bytes()
}

// fetch is a GET whose answer is bytes, not JSON.
func fetch(t *testing.T, client *http.Client, address string) (*http.Response, []byte) {
	t.Helper()

	res, err := client.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	return res, body
}

// The organisation's app logo is a file uploaded through the panel: the bytes
// decide what it is, the panel's header and the sign-in pages load it by
// version, and removing it goes back to the panel's own.
func TestLiveOrganizationLogo(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	type organizationBody struct {
		Organization struct {
			LogoVersion string `json:"logo_version"`
		} `json:"organization"`
	}

	logo := pngBytes(t, 128, 128)
	upload := map[string]string{"data": base64.StdEncoding.EncodeToString(logo)}

	var uploaded organizationBody
	super.must(http.StatusOK, http.MethodPut, "/organization/logo", upload, &uploaded)

	version := uploaded.Organization.LogoVersion
	if version == "" {
		t.Fatal("after uploading the logo there is no version")
	}

	// The header reads the version from the session's own answer.
	var me organizationBody
	super.must(http.StatusOK, http.MethodGet, "/me", nil, &me)
	if me.Organization.LogoVersion != version {
		t.Errorf("/me logo version = %q, want %q", me.Organization.LogoVersion, version)
	}

	// The panel loads it behind its session, kept by its version.
	res, body := fetch(t, super.http, s.url+"/organization/logo?v="+version)
	if res.StatusCode != http.StatusOK || !bytes.Equal(body, logo) {
		t.Fatalf("the panel's logo = %d, %d bytes; want 200 and the upload", res.StatusCode, len(body))
	}
	if got := res.Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got)
	}
	if got := res.Header.Get("Cache-Control"); !strings.HasPrefix(got, "private,") || !strings.Contains(got, "immutable") {
		t.Errorf("the panel's Cache-Control = %q, want private and immutable", got)
	}
	if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}

	// Without a session the admin API's copy is refused.
	if res, _ := fetch(t, http.DefaultClient, s.url+"/organization/logo"); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("the panel's logo without a session = %d, want 401", res.StatusCode)
	}

	// The sign-in pages load it from the public API, which names it.
	var public struct {
		Organization struct {
			LogoURL string `json:"logo_url"`
		} `json:"organization"`
	}
	res, body = fetch(t, http.DefaultClient, s.root+"/api/v1/account/organization")
	if err := json.Unmarshal(body, &public); err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("public organization = %d %s", res.StatusCode, body)
	}
	if want := "/api/v1/account/organization/logo?v=" + version; public.Organization.LogoURL != want {
		t.Errorf("public logo = %q, want %q", public.Organization.LogoURL, want)
	}

	res, body = fetch(t, http.DefaultClient, s.root+public.Organization.LogoURL)
	if res.StatusCode != http.StatusOK || !bytes.Equal(body, logo) || !strings.HasPrefix(res.Header.Get("Cache-Control"), "public,") {
		t.Errorf("the public logo = %d, %d bytes, %q; want 200, the upload, public", res.StatusCode, len(body), res.Header.Get("Cache-Control"))
	}
	// Anyone can load it, so it is never sniffed into anything but a picture.
	if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("the public logo's X-Content-Type-Options = %q, want nosniff", got)
	}

	// A browser that already has this version is told so without the bytes.
	revalidate, err := http.NewRequest(http.MethodGet, s.root+"/api/v1/account/organization/logo", nil)
	if err != nil {
		t.Fatal(err)
	}
	revalidate.Header.Set("If-None-Match", res.Header.Get("ETag"))
	res, body = func() (*http.Response, []byte) {
		res, err := http.DefaultClient.Do(revalidate)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return res, body
	}()
	if res.StatusCode != http.StatusNotModified || len(body) != 0 {
		t.Errorf("revalidating the logo it has = %d with %d bytes, want 304 and none", res.StatusCode, len(body))
	}

	// Asked for by an old version, it is checked again rather than kept.
	if res, _ := fetch(t, http.DefaultClient, s.root+"/api/v1/account/organization/logo?v=old"); res.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("an old version's Cache-Control = %q, want no-cache", res.Header.Get("Cache-Control"))
	}

	// A replacement takes its place, under a new version.
	replacement := pngBytes(t, 96, 96)
	var replaced organizationBody
	super.must(http.StatusOK, http.MethodPut, "/organization/logo", map[string]string{"data": base64.StdEncoding.EncodeToString(replacement)}, &replaced)
	if replaced.Organization.LogoVersion == version || replaced.Organization.LogoVersion == "" {
		t.Errorf("the replacement's version = %q, want a new one", replaced.Organization.LogoVersion)
	}
	if _, body := fetch(t, http.DefaultClient, s.root+"/api/v1/account/organization/logo"); !bytes.Equal(body, replacement) {
		t.Errorf("after replacing, the logo is %d bytes, want the replacement", len(body))
	}
	version = replaced.Organization.LogoVersion

	// What is refused, and that a refusal changes nothing.
	for _, refused := range []struct {
		name   string
		body   any
		status int
	}{
		{"an SVG", map[string]string{"data": base64.StdEncoding.EncodeToString([]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))}, http.StatusUnsupportedMediaType},
		{"a logo too small to see", map[string]string{"data": base64.StdEncoding.EncodeToString(pngBytes(t, 4, 4))}, http.StatusBadRequest},
		{"not base64", map[string]string{"data": "not base64!"}, http.StatusBadRequest},
		{"nothing", map[string]string{}, http.StatusBadRequest},
		// Over 512 KB once decoded, and still in the body.
		{"a logo over 512 KB", map[string]string{"data": base64.StdEncoding.EncodeToString(append(append([]byte{}, logo...), make([]byte, 600<<10)...))}, http.StatusRequestEntityTooLarge},
		// Far over: the API's body limit cuts it off before it is read, and it
		// is still a logo too large, not a broken request.
		{"a logo far over 512 KB", map[string]string{"data": base64.StdEncoding.EncodeToString(make([]byte, 2<<20))}, http.StatusRequestEntityTooLarge},
	} {
		if status := super.do(http.MethodPut, "/organization/logo", refused.body, nil); status != refused.status {
			t.Errorf("%s = %d, want %d", refused.name, status, refused.status)
		}
	}

	var unchanged organizationBody
	super.must(http.StatusOK, http.MethodGet, "/organization", nil, &unchanged)
	if unchanged.Organization.LogoVersion != version {
		t.Errorf("after the refusals the version = %q, want %q", unchanged.Organization.LogoVersion, version)
	}

	// An administrator who may not change the organisation still sees its
	// logo in the header, and cannot change it.
	const email, password = "auditor@example.com", "auditor-password-1"
	super.must(http.StatusCreated, http.MethodPost, "/admins", map[string]any{
		"email": email, "first_name": "Auditor", "status": "active",
		"password": password, "confirm_password": password,
		"assignments": []map[string]any{{"role_id": super.adminRoleID("auditor")}},
	}, nil)
	auditor := s.client()
	if status := auditor.login(email, password); status != http.StatusOK {
		t.Fatalf("auditor login = %d", status)
	}

	if res, _ := fetch(t, auditor.http, s.url+"/organization/logo?v="+version); res.StatusCode != http.StatusOK {
		t.Errorf("the auditor's logo = %d, want 200", res.StatusCode)
	}
	if status := auditor.do(http.MethodPut, "/organization/logo", upload, nil); status != http.StatusForbidden {
		t.Errorf("the auditor's upload = %d, want 403", status)
	}
	if status := auditor.do(http.MethodDelete, "/organization/logo", nil, nil); status != http.StatusForbidden {
		t.Errorf("the auditor's removal = %d, want 403", status)
	}

	// Removed, it is gone everywhere.
	var removed organizationBody
	super.must(http.StatusOK, http.MethodDelete, "/organization/logo", nil, &removed)
	if removed.Organization.LogoVersion != "" {
		t.Errorf("after removing the version = %q, want none", removed.Organization.LogoVersion)
	}
	if res, _ := fetch(t, http.DefaultClient, s.root+"/api/v1/account/organization/logo"); res.StatusCode != http.StatusNotFound {
		t.Errorf("the removed logo = %d, want 404", res.StatusCode)
	}

	// The log says each change, and nothing for a removal of nothing.
	super.must(http.StatusOK, http.MethodDelete, "/organization/logo", nil, nil)

	var logs struct {
		Logs []struct {
			Action string `json:"action"`
		} `json:"logs"`
	}
	super.must(http.StatusOK, http.MethodGet, "/logs?action=organization.updated&limit=50", nil, &logs)
	if len(logs.Logs) != 3 {
		t.Errorf("organization.updated entries = %d, want two uploads and one removal", len(logs.Logs))
	}
}
