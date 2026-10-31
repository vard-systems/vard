// Package config reads the server's settings from the environment and .env, and
// refuses ones that cannot work.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"

	"vard/internal/brand"
)

type Config struct {
	Addr string

	// AdminAddr is the admin API's own listener, so it is never on the public
	// one. Keep it on an internal interface.
	AdminAddr string

	Issuer string

	AccountURL string

	AdminURL string

	// CORSOrigins are extra origins allowed to call cookie-authenticated APIs
	// with credentials. None by default: each listed origin can act as the
	// signed-in user.
	CORSOrigins []string

	SecureUserCookies  bool
	SecureAdminCookies bool

	KeyRotation time.Duration

	AuditRetention time.Duration

	AdminMFARequired bool

	RateLimit int

	// LogLevel is the least severe line the server writes: debug, info, warn
	// or error.
	LogLevel slog.Level

	// TrustedProxies are the proxies whose X-Forwarded-For is believed. Empty
	// believes no one. Behind a proxy it must be set, or every request (and the
	// rate limit) counts as the proxy's.
	TrustedProxies []string

	// SecretKey seals stored private keys and secrets. Losing it signs everyone
	// out; leaking it with the database lets anyone sign tokens.
	SecretKey string
	// PwnedPasswordsURL is the breached-password range API new passwords are
	// checked against, "" for no check.
	PwnedPasswordsURL string

	// SecretKeyPrevious is the key SecretKey replaced: it opens what it
	// sealed, and seals nothing. Empty once `secrets reseal` has run.
	SecretKeyPrevious string

	Mail Mail

	DB DB

	Redis Redis
}

func Origin(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}

const minSecretKeyLength = 32

func SecretKeyLooksWeak(key string) bool {
	distinct := map[rune]struct{}{}
	for _, r := range key {
		distinct[r] = struct{}{}
	}

	return len(key) < 43 || len(distinct) < 16
}

type Mail struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type DB struct {
	Driver string
	DSN    string
	// MigrateDSN is the schema owner's, used only to migrate; the server then
	// runs as DSN's user, which may read and write rows and nothing more.
	// Empty migrates as DSN's user.
	MigrateDSN string
	TimeZone   string
	LogQueries bool
	Migrate    bool
	MigrateDir string

	MaxConns int
	// StatementTimeout and IdleInTransactionTimeout are Postgres's own
	// limits on each connection: a query, or a transaction left open, past
	// them is ended by the database rather than held for ever.
	StatementTimeout         time.Duration
	IdleInTransactionTimeout time.Duration
}

type Redis struct {
	Host      string
	Port      int
	Username  string
	Password  string
	CacheDB   int
	SessionDB int
	Prefix    string
}

func (r Redis) Enabled() bool {
	return r.Host != ""
}

func (r Redis) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func env(setting string) string {
	return brand.EnvPrefix + setting
}

func read() (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	return v, nil
}

func Load() (Config, error) {
	v, err := read()
	if err != nil {
		return Config{}, err
	}

	v.SetDefault(env("ADDR"), ":8080")
	// The admin listener stays on loopback unless an installation says
	// otherwise; the container image overrides it.
	v.SetDefault(env("ADMIN_ADDR"), "127.0.0.1:8081")
	v.SetDefault(env("CORS_ORIGINS"), "")
	v.SetDefault(env("TRUSTED_PROXIES"), "")
	v.SetDefault(env("RATE_LIMIT"), 20)
	v.SetDefault(env("LOG_LEVEL"), "info")
	v.SetDefault(env("ADMIN_MFA"), "optional")
	v.SetDefault(env("KEY_ROTATION_DAYS"), 90)
	v.SetDefault(env("AUDIT_RETENTION_DAYS"), 365)
	v.SetDefault(env("ISSUER"), "http://localhost:8080")
	v.SetDefault(env("ADMIN_URL"), "http://localhost:8081")
	v.SetDefault(env("SMTP_PORT"), 587)
	v.SetDefault(env("SMTP_FROM"), brand.Name+" <no-reply@localhost>")
	v.SetDefault(env("DB_DRIVER"), "postgres")
	v.SetDefault(env("DB_TIMEZONE"), "UTC")
	v.SetDefault(env("DB_LOG_QUERIES"), false)
	v.SetDefault(env("DB_MIGRATE"), true)
	v.SetDefault(env("DB_MIGRATE_DIR"), "./migrations")
	v.SetDefault(env("DB_MAX_CONNS"), 25)
	v.SetDefault(env("DB_STATEMENT_TIMEOUT"), "15s")
	v.SetDefault(env("DB_IDLE_IN_TRANSACTION_TIMEOUT"), "60s")
	v.SetDefault(env("REDIS_PORT"), 6379)
	v.SetDefault(env("REDIS_CACHE_DB"), 0)
	v.SetDefault(env("REDIS_SESSION_DB"), 1)
	v.SetDefault(env("REDIS_PREFIX"), brand.RedisPrefix)

	issuer := strings.TrimRight(v.GetString(env("ISSUER")), "/")
	accountURL := strings.TrimRight(v.GetString(env("ACCOUNT_URL")), "/")
	if accountURL == "" {
		accountURL = issuer
	}
	adminURL := strings.TrimRight(v.GetString(env("ADMIN_URL")), "/")

	cfg := Config{
		Addr:               v.GetString(env("ADDR")),
		AdminAddr:          v.GetString(env("ADMIN_ADDR")),
		Issuer:             issuer,
		AccountURL:         accountURL,
		AdminURL:           adminURL,
		CORSOrigins:        splitList(v.GetString(env("CORS_ORIGINS"))),
		SecureUserCookies:  strings.HasPrefix(accountURL, "https://"),
		SecureAdminCookies: strings.HasPrefix(adminURL, "https://"),
		AdminMFARequired:   v.GetString(env("ADMIN_MFA")) == "required",
		KeyRotation:        time.Duration(v.GetInt(env("KEY_ROTATION_DAYS"))) * 24 * time.Hour,
		AuditRetention:     time.Duration(v.GetInt(env("AUDIT_RETENTION_DAYS"))) * 24 * time.Hour,
		RateLimit:          v.GetInt(env("RATE_LIMIT")),
		TrustedProxies:     splitList(v.GetString(env("TRUSTED_PROXIES"))),
		SecretKey:          v.GetString(env("SECRET_KEY")),
		SecretKeyPrevious:  v.GetString(env("SECRET_KEY_PREVIOUS")),
		PwnedPasswordsURL:  strings.TrimSpace(v.GetString(env("PWNED_PASSWORDS_URL"))),
		Mail: Mail{
			Host:     v.GetString(env("SMTP_HOST")),
			Port:     v.GetInt(env("SMTP_PORT")),
			Username: v.GetString(env("SMTP_USERNAME")),
			Password: v.GetString(env("SMTP_PASSWORD")),
			From:     v.GetString(env("SMTP_FROM")),
		},
		DB: DB{
			Driver:     v.GetString(env("DB_DRIVER")),
			DSN:        v.GetString(env("DB_DSN")),
			MigrateDSN: v.GetString(env("DB_MIGRATE_DSN")),
			TimeZone:   v.GetString(env("DB_TIMEZONE")),
			LogQueries: v.GetBool(env("DB_LOG_QUERIES")),
			Migrate:    v.GetBool(env("DB_MIGRATE")),
			MigrateDir: v.GetString(env("DB_MIGRATE_DIR")),
			MaxConns:   v.GetInt(env("DB_MAX_CONNS")),

			StatementTimeout:         v.GetDuration(env("DB_STATEMENT_TIMEOUT")),
			IdleInTransactionTimeout: v.GetDuration(env("DB_IDLE_IN_TRANSACTION_TIMEOUT")),
		},
		Redis: Redis{
			Host:      strings.TrimSpace(v.GetString(env("REDIS_HOST"))),
			Port:      v.GetInt(env("REDIS_PORT")),
			Username:  v.GetString(env("REDIS_USERNAME")),
			Password:  v.GetString(env("REDIS_PASSWORD")),
			CacheDB:   v.GetInt(env("REDIS_CACHE_DB")),
			SessionDB: v.GetInt(env("REDIS_SESSION_DB")),
			Prefix:    v.GetString(env("REDIS_PREFIX")),
		},
	}

	if v.IsSet(env("SECURE_COOKIES")) && v.GetString(env("SECURE_COOKIES")) != "" {
		cfg.SecureUserCookies = v.GetBool(env("SECURE_COOKIES"))
		cfg.SecureAdminCookies = cfg.SecureUserCookies
	}

	// No default for the DSN: it names the database and carries the password,
	// so a missing one should stop the server, not quietly connect somewhere.
	if cfg.DB.DSN == "" {
		return Config{}, errors.New(env("DB_DSN") + " is not set")
	}

	// No default for the secret key either: a default would be the same key
	// on every installation, which is no secret at all.
	if len(cfg.SecretKey) < minSecretKeyLength {
		return Config{}, fmt.Errorf("%s must be at least %d characters; make one with: openssl rand -base64 32", env("SECRET_KEY"), minSecretKeyLength)
	}

	urls := map[string]string{
		env("ISSUER"):      cfg.Issuer,
		env("ACCOUNT_URL"): cfg.AccountURL,
		env("ADMIN_URL"):   cfg.AdminURL,
	}
	for name, value := range urls {
		if parsed, err := url.Parse(value); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return Config{}, fmt.Errorf("%s must be an http or https URL, got %q", name, value)
		}
	}

	// One listener for both would put the admin API back on the public one,
	// which is exactly what the second listener is there to prevent.
	if cfg.AdminAddr == "" || cfg.AdminAddr == cfg.Addr {
		return Config{}, errors.New(env("ADMIN_ADDR") + " must be set, and differ from " + env("ADDR") + ": the admin API has its own listener")
	}

	if mode := v.GetString(env("ADMIN_MFA")); mode != "required" && mode != "optional" {
		return Config{}, fmt.Errorf("%s must be required or optional, got %q", env("ADMIN_MFA"), mode)
	}

	if cfg.KeyRotation < 0 {
		return Config{}, errors.New(env("KEY_ROTATION_DAYS") + " must be zero or more")
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(v.GetString(env("LOG_LEVEL")))); err != nil {
		return Config{}, fmt.Errorf("%s must be debug, info, warn or error, got %q", env("LOG_LEVEL"), v.GetString(env("LOG_LEVEL")))
	}

	if cfg.RateLimit < 0 {
		return Config{}, errors.New(env("RATE_LIMIT") + " must be zero or more")
	}

	if cfg.AuditRetention < 0 {
		return Config{}, errors.New(env("AUDIT_RETENTION_DAYS") + " must be zero or more")
	}

	if cfg.DB.MaxConns < 1 {
		return Config{}, errors.New(env("DB_MAX_CONNS") + " must be one or more")
	}
	if cfg.DB.StatementTimeout < 0 || cfg.DB.IdleInTransactionTimeout < 0 {
		return Config{}, fmt.Errorf("%s and %s must be zero (no limit) or more",
			env("DB_STATEMENT_TIMEOUT"), env("DB_IDLE_IN_TRANSACTION_TIMEOUT"))
	}

	if cfg.Redis.Enabled() {
		if cfg.Redis.Port < 1 || cfg.Redis.Port > 65535 {
			return Config{}, fmt.Errorf("%s must be a port number, got %d", env("REDIS_PORT"), cfg.Redis.Port)
		}
		if cfg.Redis.CacheDB < 0 || cfg.Redis.SessionDB < 0 {
			return Config{}, fmt.Errorf("%s and %s must be zero or more", env("REDIS_CACHE_DB"), env("REDIS_SESSION_DB"))
		}
		// One database for both would let flushing the cache sign everybody
		// out, which is what keeping them apart is for.
		if cfg.Redis.CacheDB == cfg.Redis.SessionDB {
			return Config{}, fmt.Errorf("%s and %s must be different databases, both are %d",
				env("REDIS_CACHE_DB"), env("REDIS_SESSION_DB"), cfg.Redis.CacheDB)
		}
	}

	return cfg, nil
}

func splitList(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
