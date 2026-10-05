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

	"github.com/matteotorelli/risma/backend/internal/auth"
	"github.com/matteotorelli/risma/backend/internal/config"
	"github.com/matteotorelli/risma/backend/internal/dashboard"
	"github.com/matteotorelli/risma/backend/internal/database"
	"github.com/matteotorelli/risma/backend/internal/httpx"
	"github.com/matteotorelli/risma/backend/internal/mail"
	"github.com/matteotorelli/risma/backend/internal/product"
	"github.com/matteotorelli/risma/backend/internal/role"
	"github.com/matteotorelli/risma/backend/internal/user"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("loading config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		logger.Error("connecting to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	smtpSender, err := mail.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	if err != nil {
		logger.Error("creating smtp sender", "error", err)
		os.Exit(1)
	}
	mailSvc := mail.NewService(smtpSender, mail.NewRepository(db), logger)

	userRepo := user.NewRepository(db)

	authSvc, err := auth.NewService(userRepo, auth.NewRepository(db), mailSvc, cfg.JWTSecret, cfg.AppBaseURL)
	if err != nil {
		logger.Error("creating auth service", "error", err)
		os.Exit(1)
	}
	authMW := auth.NewMiddleware(authSvc)

	roleSvc := role.NewService(role.NewRepository(db))
	userSvc := user.NewService(userRepo, roleSvc, authSvc)
	productSvc := product.NewService(product.NewRepository(db), userRepo, mailSvc, logger)
	dashboardSvc := dashboard.NewService(dashboard.NewRepository(db))

	// Le rotte autenticate richiedono anche il token CSRF sulle richieste
	// che modificano dati; l'ordine dà 401 senza sessione, poi 403.
	requireAuth := func(next http.Handler) http.Handler {
		return authMW.RequireAuth(authMW.RequireCSRF(next))
	}
	requireAdmin := func(next http.Handler) http.Handler {
		return requireAuth(authMW.RequireRole(role.Admin)(next))
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Limiti per IP sulle rotte pubbliche, oltre al blocco dell'account dopo
	// 5 tentativi: rallentano chi prova molti username o invia molte email.
	attemptLimiter := httpx.NewRateLimiter(10, time.Minute, cfg.TrustedProxies)
	emailLimiter := httpx.NewRateLimiter(5, 15*time.Minute, cfg.TrustedProxies)
	go attemptLimiter.Run(ctx, 5*time.Minute)
	go emailLimiter.Run(ctx, 5*time.Minute)

	auth.NewHandler(authSvc, logger, cfg.TrustedProxies).RegisterRoutes(mux, authMW, auth.RateLimits{
		Attempts: attemptLimiter.Limit,
		Emails:   emailLimiter.Limit,
	})
	role.NewHandler(roleSvc, logger).RegisterRoutes(mux, requireAuth)
	user.NewHandler(userSvc, logger).RegisterRoutes(mux, requireAuth, requireAdmin)
	product.NewHandler(productSvc, logger).RegisterRoutes(mux, requireAuth)
	dashboard.NewHandler(dashboardSvc, logger).RegisterRoutes(mux, requireAuth)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           httpx.Recovery(logger)(httpx.Logging(logger)(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	logger.Info("server listening", "addr", srv.Addr)

	<-ctx.Done()

	logger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	// Le email in background usano il database: si attende la loro fine
	// prima che il defer chiuda il pool.
	if err := mailSvc.Wait(shutdownCtx); err != nil {
		logger.Error("waiting for background emails", "error", err)
	}
}
