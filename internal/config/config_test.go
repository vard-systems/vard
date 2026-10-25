package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vard/internal/brand"
)

// clearEnv unsets every VARD_ variable for the duration of the test, so a
// developer's own environment cannot change the result. Viper treats an empty
// variable as unset.
func clearEnv(t *testing.T) {
	t.Helper()

	for _, entry := range os.Environ() {
		if name, _, found := strings.Cut(entry, "="); found && strings.HasPrefix(name, "VARD_") {
			t.Setenv(name, "")
		}
	}
}

// writeEnv puts a .env in a temporary directory and makes it the working
// directory, because Load reads ./.env.
func writeEnv(t *testing.T, contents string) {
	t.Helper()

	clearEnv(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
}

func TestLoadReadsEnvFile(t *testing.T) {
	writeEnv(t, `
VARD_ADDR=:9000
VARD_CORS_ORIGINS=http://a.test, http://b.test
VARD_DB_DSN=postgres://user:pw@localhost:5432/mydb
VARD_SECRET_KEY=0123456789abcdef0123456789abcdef
VARD_ISSUER=https://auth.example.com/
VARD_DB_LOG_QUERIES=true
VARD_DB_MIGRATE=false
VARD_SECURE_COOKIES=true
VARD_TRUSTED_PROXIES=10.0.0.1, 192.168.0.0/16
`)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.SecureUserCookies || !cfg.SecureAdminCookies {
		t.Error("VARD_SECURE_COOKIES=true did not turn both cookies secure")
	}
	if len(cfg.TrustedProxies) != 2 || cfg.TrustedProxies[1] != "192.168.0.0/16" {
		t.Errorf("TrustedProxies = %v, want both, trimmed", cfg.TrustedProxies)
	}

	if cfg.Issuer != "https://auth.example.com" {
		t.Errorf("Issuer = %q, want it without the trailing slash", cfg.Issuer)
	}

	if cfg.Addr != ":9000" {
		t.Errorf("Addr = %q, want :9000", cfg.Addr)
	}
	if cfg.DB.DSN != "postgres://user:pw@localhost:5432/mydb" {
		t.Errorf("DSN = %q", cfg.DB.DSN)
	}
	if !cfg.DB.LogQueries {
		t.Error("LogQueries = false, want true")
	}
	if cfg.DB.Migrate {
		t.Error("Migrate = true, want false")
	}

	want := []string{"http://a.test", "http://b.test"}
	if len(cfg.CORSOrigins) != len(want) {
		t.Fatalf("CORSOrigins = %v, want %v", cfg.CORSOrigins, want)
	}
	for i, origin := range want {
		if cfg.CORSOrigins[i] != origin {
			t.Errorf("CORSOrigins[%d] = %q, want %q (spaces should be trimmed)", i, cfg.CORSOrigins[i], origin)
		}
	}
}

func TestLoadUsesDefaults(t *testing.T) {
	writeEnv(t, "VARD_DB_DSN=postgres://u:p@localhost:5432/db\nVARD_SECRET_KEY=0123456789abcdef0123456789abcdef\n")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want the default :8080", cfg.Addr)
	}
	if cfg.DB.Driver != "postgres" {
		t.Errorf("Driver = %q, want the default postgres", cfg.DB.Driver)
	}
	if cfg.DB.TimeZone != "Asia/Bishkek" {
		t.Errorf("TimeZone = %q, want the default Asia/Bishkek", cfg.DB.TimeZone)
	}
	if !cfg.DB.Migrate {
		t.Error("Migrate = false, want the default true")
	}
	if cfg.Issuer != "http://localhost:8080" || cfg.AccountURL != cfg.Issuer || cfg.AdminURL != "http://localhost:8081" {
		t.Errorf("Issuer, AccountURL, AdminURL = %q, %q, %q, want the local defaults", cfg.Issuer, cfg.AccountURL, cfg.AdminURL)
	}
	if cfg.AdminAddr != "127.0.0.1:8081" || len(cfg.CORSOrigins) != 0 || cfg.RateLimit != 20 {
		t.Errorf("AdminAddr, CORSOrigins, RateLimit = %q, %v, %d, want 127.0.0.1:8081, none, 20", cfg.AdminAddr, cfg.CORSOrigins, cfg.RateLimit)
	}
	if cfg.SecureUserCookies || cfg.SecureAdminCookies {
		t.Error("cookies are secure on the http defaults, which a browser would not send back")
	}
	// A fresh installation does not make anybody set up an authenticator:
	// each administrator turns one on for themselves, and a super admin can
	// require them of everybody once there is somebody to ask.
	if cfg.AdminMFARequired {
		t.Error("AdminMFARequired = true, want two-factor sign-in optional by default")
	}
	if cfg.Mail.Host != "" {
		t.Errorf("Mail.Host = %q, want none: email is logged by default", cfg.Mail.Host)
	}
	if cfg.DB.MigrateDir != "./migrations" {
		t.Errorf("MigrateDir = %q, want the default ./migrations", cfg.DB.MigrateDir)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want the default info", cfg.LogLevel)
	}
	if cfg.AuditRetention != 365*24*time.Hour || cfg.DB.MaxConns != 25 {
		t.Errorf("AuditRetention, MaxConns = %v, %d, want a year and 25", cfg.AuditRetention, cfg.DB.MaxConns)
	}
}

// What would leave the server unable to reach its database, or sweeping the
// activity log backwards, stops it at startup.
func TestLoadRefusesLimitsThatCannotWork(t *testing.T) {
	const base = "VARD_DB_DSN=postgres://localhost/x\nVARD_SECRET_KEY=0123456789abcdef0123456789abcdef\n"

	tests := []struct {
		name    string
		env     string
		wantErr bool
	}{
		{name: "an activity log kept forever", env: base + "VARD_AUDIT_RETENTION_DAYS=0\n"},
		{name: "a retention that is negative", env: base + "VARD_AUDIT_RETENTION_DAYS=-1\n", wantErr: true},
		{name: "one connection", env: base + "VARD_DB_MAX_CONNS=1\n"},
		{name: "no connections at all", env: base + "VARD_DB_MAX_CONNS=0\n", wantErr: true},
		{name: "a log level in any case", env: base + "VARD_LOG_LEVEL=Debug\n"},
		{name: "a log level slog does not have", env: base + "VARD_LOG_LEVEL=verbose\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeEnv(t, tt.env)

			if _, err := Load(); (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, want an error: %v", err, tt.wantErr)
			}
		})
	}
}

// The environment has to win over .env, or a container could not change a
// single setting without rewriting the file.
func TestEnvironmentOverridesEnvFile(t *testing.T) {
	writeEnv(t, "VARD_ADDR=:8080\nVARD_DB_DSN=postgres://u:p@localhost:5432/from_file\nVARD_SECRET_KEY=0123456789abcdef0123456789abcdef\n")
	t.Setenv("VARD_ADDR", ":7777")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Addr != ":7777" {
		t.Errorf("Addr = %q, want :7777 from the environment", cfg.Addr)
	}
}

// The secret key encrypts the signing keys, so a missing or short one has to
// stop the server rather than fall back to something guessable.
func TestLoadFailsWithoutSecretKey(t *testing.T) {
	writeEnv(t, "VARD_DB_DSN=postgres://u:p@localhost:5432/db\n")
	if _, err := Load(); err == nil {
		t.Fatal("want an error when VARD_SECRET_KEY is missing, got none")
	}

	writeEnv(t, "VARD_DB_DSN=postgres://u:p@localhost:5432/db\nVARD_SECRET_KEY=short\n")
	if _, err := Load(); err == nil {
		t.Fatal("want an error when VARD_SECRET_KEY is too short, got none")
	}
}

// Deployed on https, the session cookies are Secure without being told, and
// the account app defaults to the issuer's origin.
func TestLoadDerivesFromPublicURLs(t *testing.T) {
	writeEnv(t, `
VARD_DB_DSN=postgres://u:p@localhost:5432/db
VARD_SECRET_KEY=0123456789abcdef0123456789abcdef
VARD_ISSUER=https://id.mywebsite.com
VARD_ADMIN_URL=https://admin-id.mywebsite.com
`)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.AccountURL != "https://id.mywebsite.com" {
		t.Errorf("AccountURL = %q, want the issuer", cfg.AccountURL)
	}
	if !cfg.SecureUserCookies || !cfg.SecureAdminCookies {
		t.Error("cookies are not secure on https")
	}
	if Origin("https://id.mywebsite.com/login?x=1") != "https://id.mywebsite.com" {
		t.Error("Origin did not keep only the scheme and host")
	}
}

// Served over plain http, a Secure cookie would never be sent back, so the
// sessions are not marked Secure.
func TestLoadKeepsCookiesPlainOnHTTP(t *testing.T) {
	writeEnv(t, `
VARD_DB_DSN=postgres://u:p@localhost:5432/db
VARD_SECRET_KEY=0123456789abcdef0123456789abcdef
VARD_ISSUER=http://localhost:5173
VARD_ADMIN_URL=http://localhost:5174
`)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.SecureUserCookies || cfg.SecureAdminCookies {
		t.Error("cookies are secure on plain http")
	}
}

// The admin API must never share the public listener.
func TestLoadRefusesOneListenerForBoth(t *testing.T) {
	writeEnv(t, "VARD_DB_DSN=postgres://u:p@localhost:5432/db\nVARD_SECRET_KEY=0123456789abcdef0123456789abcdef\nVARD_ADDR=:9000\nVARD_ADMIN_ADDR=:9000\n")

	if _, err := Load(); err == nil {
		t.Fatal("want an error when the admin listener is the public one")
	}
}

func TestLoadFailsWithoutDSN(t *testing.T) {
	writeEnv(t, "VARD_ADDR=:8080\n")

	if _, err := Load(); err == nil {
		t.Fatal("want an error when VARD_DB_DSN is missing, got none")
	}
}

// A missing .env is normal in a container, where only real environment
// variables are set.
func TestLoadWithoutEnvFile(t *testing.T) {
	clearEnv(t)
	t.Chdir(t.TempDir())
	t.Setenv("VARD_DB_DSN", "postgres://u:p@localhost:5432/db")
	t.Setenv("VARD_SECRET_KEY", "0123456789abcdef0123456789abcdef")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want the default :8080", cfg.Addr)
	}
}

// Redis is optional: without a host every read goes to the database, which
// is how the integration tests and a bare checkout run.
func TestLoadRedis(t *testing.T) {
	const base = "VARD_DB_DSN=postgres://localhost/x\nVARD_SECRET_KEY=0123456789abcdef0123456789abcdef\n"

	tests := []struct {
		name    string
		env     string
		want    Redis
		wantErr bool
	}{
		{
			name: "no host is no Redis",
			env:  base,
			want: Redis{Port: 6379, SessionDB: 1, Prefix: brand.RedisPrefix},
		},
		{
			name: "everything named",
			env: base + "VARD_REDIS_HOST=localhost\nVARD_REDIS_PORT=6380\nVARD_REDIS_USERNAME=app\n" +
				"VARD_REDIS_PASSWORD=pw\nVARD_REDIS_CACHE_DB=3\nVARD_REDIS_SESSION_DB=4\nVARD_REDIS_PREFIX=staging:\n",
			want: Redis{Host: "localhost", Port: 6380, Username: "app", Password: "pw", CacheDB: 3, SessionDB: 4, Prefix: "staging:"},
		},
		{
			name:    "a port that is not one",
			env:     base + "VARD_REDIS_HOST=localhost\nVARD_REDIS_PORT=70000\n",
			wantErr: true,
		},
		{
			name:    "a database before the first",
			env:     base + "VARD_REDIS_HOST=localhost\nVARD_REDIS_CACHE_DB=-1\n",
			wantErr: true,
		},
		{
			name:    "one database for the cache and the sessions",
			env:     base + "VARD_REDIS_HOST=localhost\nVARD_REDIS_CACHE_DB=2\nVARD_REDIS_SESSION_DB=2\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeEnv(t, tt.env)

			cfg, err := Load()
			switch {
			case tt.wantErr && err == nil:
				t.Fatalf("Load() = nothing, want an error")
			case tt.wantErr:
				return
			case err != nil:
				t.Fatal(err)
			}

			if cfg.Redis != tt.want {
				t.Errorf("Redis = %+v, want %+v", cfg.Redis, tt.want)
			}
			if cfg.Redis.Enabled() != (tt.want.Host != "") {
				t.Errorf("Enabled() = %v", cfg.Redis.Enabled())
			}
		})
	}
}
