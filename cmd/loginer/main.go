// Command loginer runs the server: the provider and account API on one listener,
// the admin API on the other, until it is told to stop.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"loginer/i18n"
	"loginer/internal/api"
	"loginer/internal/brand"
	"loginer/internal/cache"
	"loginer/internal/config"
	"loginer/internal/database"
	"loginer/internal/jose"
	"loginer/internal/mail"
	"loginer/internal/oidc"
	"loginer/internal/store"
)

// version is set at build time (-ldflags "-X main.version=…", from
// `git describe`, which names the commit too).
var version = "dev"

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Gin defaults to debug mode, which a bare binary under systemd would run
	// in production; GIN_MODE=debug still turns it on.
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := run(log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	log.Info(brand.Name+" starting", "version", version)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if config.SecretKeyLooksWeak(cfg.SecretKey) {
		log.Warn("LOGINER_SECRET_KEY does not look like 32 random bytes; a low-entropy key can be brute-forced against a database dump to recover sealed secrets — generate one with: openssl rand -base64 32")
	}

	db, err := database.Open(cfg.DB)
	if err != nil {
		return err
	}
	defer database.Close(db)

	log.Info("database connected")

	if cfg.DB.Migrate {
		if err := database.Migrate(db, cfg.DB, log); err != nil {
			return err
		}
	}

	// A configured Redis that does not answer stops startup; none configured
	// runs without one.
	shared, err := cache.Open(context.Background(), cfg.Redis, log)
	if err != nil {
		return err
	}
	defer shared.Close()

	if shared != nil {
		if err := shared.BumpAuthorityGroups(context.Background()); err != nil {
			log.Warn("could not bump the cache's authority groups at startup", "error", err)
		}

		log.Info("redis connected", "addr", cfg.Redis.Addr(), "cache_db", cfg.Redis.CacheDB,
			"session_db", cfg.Redis.SessionDB, "prefix", cfg.Redis.Prefix)
	} else {
		log.Info("redis is not configured; reading every page from the database")
	}

	st := store.New(db).WithCache(shared)

	sealer, err := jose.NewSealer(cfg.SecretKey)
	if err != nil {
		return err
	}

	if err := st.EnsureAdminSecurity(context.Background(), cfg.AdminMFARequired); err != nil {
		return err
	}
	if err := ensureMailSettings(context.Background(), st, sealer, cfg.Mail); err != nil {
		return err
	}
	if err := st.EnsureOTPSettings(context.Background()); err != nil {
		return err
	}

	shipped, err := i18n.Shipped()
	if err != nil {
		return err
	}
	if err := st.EnsureLanguages(context.Background(), shipped); err != nil {
		return err
	}

	if err := st.EnsureSystemAPIs(context.Background()); err != nil {
		return err
	}

	mailer := mail.New(mail.FromStore(st, sealer), log)

	// Load or create signing keys before serving, so a wrong LOGINER_SECRET_KEY
	// stops startup here.
	provider, err := oidc.New(context.Background(), cfg, st, mailer, log)
	if err != nil {
		return err
	}

	public, err := api.NewPublic(cfg, log, provider, shared)
	if err != nil {
		return err
	}

	admin, err := api.NewAdmin(cfg, st, log, provider, shared)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go provider.MaintainKeys(ctx)
	go st.KeepSwept(ctx, cfg.AuditRetention, log)
	go shared.KeepSettling(ctx)

	return serve(ctx, log, map[string]*http.Server{
		"public": newServer(cfg.Addr, public),
		"admin":  newServer(cfg.AdminAddr, admin),
	})
}

func ensureMailSettings(ctx context.Context, st *store.Store, sealer *jose.Sealer, cfg config.Mail) error {
	settings, password := mail.Seed(cfg)

	if password != "" {
		sealed, err := sealer.SealBytes([]byte(password))
		if err != nil {
			return err
		}
		settings.Password = sealed
	}

	return st.EnsureMailSettings(ctx, settings)
}

// Connection timeouts, so a slow client cannot hold a goroutine forever. Bodies
// are small, so header and read limits are short; writes are long because a
// response may wait on an identity provider's endpoints or a mail server.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 2 * time.Minute
	idleTimeout       = 2 * time.Minute
)

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

const shutdownGrace = 15 * time.Second

// serve runs every server until ctx ends or one of them fails, then gives the
// requests under way shutdownGrace to finish.
func serve(ctx context.Context, log *slog.Logger, servers map[string]*http.Server) error {
	failed := make(chan error, len(servers))
	for name, server := range servers {
		log.Info("server listening", "server", name, "addr", server.Addr)

		go func() {
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				failed <- fmt.Errorf("the %s server: %w", name, err)
			}
		}()
	}

	var err error
	select {
	case err = <-failed:
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdown, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()

	for _, server := range servers {
		_ = server.Shutdown(shutdown)
	}

	return err
}
