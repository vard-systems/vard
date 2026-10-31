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
	"github.com/google/uuid"

	"vard/internal/application/auth"
	"vard/internal/application/oidc"
	"vard/internal/brand"
	"vard/internal/config"
	"vard/internal/domain/model"
	"vard/internal/infrastructure/cache"
	"vard/internal/infrastructure/database"
	"vard/internal/infrastructure/mail"
	"vard/internal/infrastructure/store"
	"vard/internal/interfaces/api"
	"vard/internal/jose"
)

// serveCommand runs the server until it is told to stop.
func serveCommand() int {
	// The level is the configuration's, which is read inside run: until then,
	// and for a configuration that does not load, it is info.
	level := new(slog.LevelVar)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
		With("service", brand.Slug)

	// Gin defaults to debug mode, which a bare binary under systemd would run
	// in production; GIN_MODE=debug still turns it on.
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := run(log, level); err != nil {
		log.Error("fatal", "error", err)
		return 1
	}
	return 0
}

func run(log *slog.Logger, level *slog.LevelVar) error {
	log.Info("starting", "version", version)

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	level.Set(cfg.LogLevel)

	if config.SecretKeyLooksWeak(cfg.SecretKey) {
		log.Warn("VARD_SECRET_KEY does not look like 32 random bytes; a low-entropy key can be brute-forced against a database dump to recover sealed secrets — generate one with: openssl rand -base64 32")
	}

	db, err := database.Connect(cfg.DB, log)
	if err != nil {
		return err
	}
	defer database.Close(db)

	log.Info("database connected")

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

	sealer, err := jose.NewSealer(cfg.SecretKey, cfg.SecretKeyPrevious)
	if err != nil {
		return err
	}

	// Mid-rotation, everything the previous key sealed is sealed again under
	// the current one before anything is served, so the previous key can be
	// dropped as soon as this says so.
	if cfg.SecretKeyPrevious != "" {
		resealed, err := st.ResealAll(context.Background(), resealWith(sealer))
		if err != nil {
			return fmt.Errorf("sealing stored secrets under the new key: %w", err)
		}
		log.Warn("every stored secret is sealed under VARD_SECRET_KEY now; remove VARD_SECRET_KEY_PREVIOUS and restart",
			"resealed", resealed)
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

	if err := st.EnsureSystemAPIs(context.Background()); err != nil {
		return err
	}

	// Until someone is an administrator, setting the panel up takes a token
	// only the server's log shows, not merely reaching the console first.
	setupToken, err := auth.IssueSetupToken(context.Background(), st, time.Now())
	if err != nil {
		return err
	}
	if setupToken != "" {
		log.Warn("no administrator exists yet: open the setup page and enter this one-time setup token",
			"url", cfg.AdminURL+auth.SetupPage, "setup_token", setupToken,
			"expires_in", model.SetupTokenLifetime.String())
	}

	mailer := mail.New(mail.FromStore(st, sealer), log)

	// Load or create signing keys before serving, so a wrong VARD_SECRET_KEY
	// stops startup here.
	provider, err := oidc.New(context.Background(), cfg, st, sealer, mailer, log)
	if err != nil {
		return err
	}

	public, err := api.NewPublic(cfg, log, provider, shared)
	if err != nil {
		return err
	}

	admin, err := api.NewAdmin(cfg, st, sealer, log, provider, shared)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go provider.MaintainKeys(ctx)
	go st.KeepSwept(ctx, cfg.AuditRetention, log)
	go shared.KeepSettling(ctx)

	err = serve(ctx, log, map[string]*http.Server{
		"public": newServer(cfg.Addr, public),
		"admin":  newServer(cfg.AdminAddr, admin),
	})

	// The servers are stopped; logouts they started are still being told.
	waiting, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	provider.Wait(waiting)

	return err
}

func ensureMailSettings(ctx context.Context, st *store.Store, sealer *jose.Sealer, cfg config.Mail) error {
	settings, password := mail.Seed(cfg)

	// Its id before it is stored, so the password is sealed to it.
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	settings.ID = id

	if password != "" {
		sealed, err := sealer.SealFor(settings.PasswordWhere(), []byte(password))
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
