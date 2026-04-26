package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"

	iaminternal "github.com/mainframe/tm-system/iam/internal"
	"github.com/mainframe/tm-system/iam/internal/middleware"
	"github.com/mainframe/tm-system/iam/internal/repository"
	iamsvc "github.com/mainframe/tm-system/iam/internal/service"
)

// IAMConfig extends BaseConfig with IAM-specific settings.
type IAMConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	HTTP              HTTPConfig  `mapstructure:"http"`
	IAM               IAMSettings `mapstructure:"iam"`
}

// HTTPConfig holds the HTTP server port.
type HTTPConfig struct {
	Port int `mapstructure:"port"`
}

// IAMSettings holds IAM-specific configuration.
type IAMSettings struct {
	JWTSecret     string `mapstructure:"jwt_secret"     json:"-"`
	AdminPassword string `mapstructure:"admin_password" json:"-"`
}

// Run starts the IAM HTTP server and blocks until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	var cfg IAMConfig
	if err := config.Load(configPath, &cfg); err != nil {
		return fmt.Errorf("iam: config error: %w", err)
	}

	level := logging.ParseLevel(cfg.Service.LogLevel)
	logger := logging.NewLogger(cfg.Service.Name, level)

	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		return fmt.Errorf("iam: failed to open SQLite: %w", err)
	}
	defer sdb.Close()
	logger.Info("using SQLite database", "path", cfg.SQLite.Path)

	// Repositories.
	userRepo := repository.NewUserRepository(sdb)
	roleRepo := repository.NewRoleRepository(sdb)
	tokenRepo := repository.NewTokenRepository(sdb)

	// Periodic expired-token cleanup (replaces MongoDB TTL index).
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := tokenRepo.DeleteExpired(ctx); err != nil {
					logger.Warn("failed to delete expired tokens", "error", err)
				}
			}
		}
	}()

	// Services.
	authSvc := iamsvc.NewAuthService(ctx, userRepo, tokenRepo, cfg.IAM.JWTSecret)
	userSvc := iamsvc.NewUserService(userRepo, roleRepo)
	roleSvc := iamsvc.NewRoleService(roleRepo)

	// Seed default roles + admin user on first startup.
	if err := iamsvc.Seed(ctx, userRepo, roleRepo, cfg.IAM.AdminPassword, logger); err != nil {
		return fmt.Errorf("iam: seed error: %w", err)
	}

	// Load role documents so stored endpoint permissions are added to Casbin.
	allRoles, err := roleSvc.ListRaw(ctx)
	if err != nil {
		return fmt.Errorf("iam: failed to load roles for Casbin init: %w", err)
	}

	authzEnforcer, err := middleware.NewIAMEnforcer(allRoles)
	if err != nil {
		return fmt.Errorf("iam: casbin enforcer init: %w", err)
	}

	permSvc := iamsvc.NewPermissionService(authzEnforcer, roleRepo)

	// HTTP router.
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"iam"}`)) //nolint:errcheck
	})

	iaminternal.RegisterRoutes(r, authSvc, userSvc, roleSvc, permSvc, authzEnforcer, logger)

	addr := fmt.Sprintf(":%d", cfg.HTTP.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("IAM service listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down IAM service")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("iam: HTTP server shutdown error: %w", err)
	}
	logger.Info("IAM service stopped gracefully")
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
