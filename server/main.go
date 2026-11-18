// Command server runs the BrainLogistics tracking API (customer-facing and admin).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/api"
	"github.com/KyAnhVo/brainlog-website/server/internal/config"
	"github.com/KyAnhVo/brainlog-website/server/internal/seed"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
	"github.com/KyAnhVo/brainlog-website/server/internal/store/memstore"
	"github.com/KyAnhVo/brainlog-website/server/internal/store/pgstore"
	"github.com/KyAnhVo/brainlog-website/server/internal/web"
)

func main() {
	var err error
	if len(os.Args) > 1 && os.Args[1] == "create-account" {
		err = createAccount(os.Args[2:])
	} else {
		err = run()
	}
	if err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// openStore opens the configured store. For Postgres this also applies pending migrations.
// The returned close function is always safe to call.
func openStore(ctx context.Context, cfg config.Config) (store.Store, func(), error) {
	if cfg.Store == config.StorePostgres {
		pg, err := pgstore.Open(ctx, cfg.DBAddr)
		if err != nil {
			return nil, func() {}, err
		}
		return pg, pg.Close, nil
	}
	return memstore.New(), func() {}, nil
}

// routes splits the single host into three areas:
//
//	/healthz    liveness check (200 "ok")
//	/api/...    the JSON API (customer: /api/tracking/..., admin: /api/admin/...)
//	/admin/...  the admin panel SPA ("/admin" redirects to "/admin/")
//	/...        the customer SPA
func routes(apiHandler, adminSPA, customerSPA http.Handler) http.Handler {
	mux := http.NewServeMux()
	// Liveness for the platform's health checks. Deliberately does not touch the database:
	// a slow Neon wake-up must not get healthy instances restarted.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte("ok\n"))
	})
	mux.Handle("/api/", apiHandler)
	mux.Handle("/admin/", http.StripPrefix("/admin", adminSPA))
	mux.Handle("/", customerSPA)
	return mux
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, closeStore, err := openStore(ctx, cfg)
	defer closeStore()
	if err != nil {
		return err
	}

	if *cfg.SeedMockData {
		loaded, err := seed.Load(ctx, st)
		if err != nil {
			return err
		}
		if loaded {
			log.Warn("loaded mock data with publicly known passwords (see devnote/server-mock.md)", "store", cfg.Store)
		} else {
			log.Info("mock data already present, not seeding again")
		}
	}

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: routes(
			api.New(api.Config{Store: st, CookieSecure: cfg.CookieSecure, Logger: log}),
			web.Admin(),
			web.Customer(),
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		// DB_ADDR is deliberately not logged: it contains the database password.
		log.Info("listening", "addr", cfg.Addr, "appEnv", cfg.AppEnv, "store", cfg.Store, "cookieSecure", cfg.CookieSecure)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}
	return nil
}
