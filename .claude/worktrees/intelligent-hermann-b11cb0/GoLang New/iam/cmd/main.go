package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"

	iam "github.com/mainframe/tm-system/iam/internal"
	"github.com/mainframe/tm-system/iam/internal/middleware"
	"github.com/mainframe/tm-system/iam/internal/repository"
	"github.com/mainframe/tm-system/iam/internal/service"
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
	JWTSecret     string `mapstructure:"jwt_secret"      json:"-"`
	AdminPassword string `mapstructure:"admin_password"  json:"-"`
}

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	var cfg IAMConfig
	if err := config.Load(configPath, &cfg); err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	level := logging.ParseLevel(cfg.Service.LogLevel)
	logger := logging.NewLogger(cfg.Service.Name, level)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Validate required secrets before doing anything else.
	if cfg.IAM.JWTSecret == "" {
		logger.Error("jwt_secret must not be empty — set it in config or via environment variable")
		os.Exit(1)
	}
	if len(cfg.IAM.JWTSecret) < 32 {
		logger.Warn("jwt_secret is shorter than 32 characters; consider using a longer secret for better security")
	}
	if cfg.IAM.AdminPassword == "" {
		logger.Error("admin_password must not be empty — set it in config or via environment variable")
		os.Exit(1)
	}

	// Open SQLite database.
	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		logger.Error("failed to open SQLite", "error", err)
		os.Exit(1)
	}
	defer sdb.Close()
	logger.Info("using SQLite database", "path", cfg.SQLite.Path)

	// Build repositories.
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

	// Build services.
	authSvc := service.NewAuthService(ctx, userRepo, tokenRepo, cfg.IAM.JWTSecret)
	userSvc := service.NewUserService(userRepo, roleRepo)
	roleSvc := service.NewRoleService(roleRepo)

	// Seed default roles and admin user on first startup.
	if err := service.Seed(ctx, userRepo, roleRepo, cfg.IAM.AdminPassword, logger); err != nil {
		logger.Error("failed to seed database", "error", err)
		os.Exit(1)
	}

	// Load all roles so their stored endpoint permissions can be added to Casbin.
	allRoles, err := roleSvc.ListRaw(ctx)
	if err != nil {
		logger.Error("failed to load roles for Casbin init", "error", err)
		os.Exit(1)
	}

	authzEnforcer, err := middleware.NewIAMEnforcer(allRoles)
	if err != nil {
		logger.Error("failed to initialize casbin enforcer", "error", err)
		os.Exit(1)
	}

	permSvc := service.NewPermissionService(authzEnforcer, roleRepo)

	// Build HTTP router.
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"iam"}`)) //nolint:errcheck
	})

	iam.RegisterRoutes(r, authSvc, userSvc, roleSvc, permSvc, authzEnforcer, logger)

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
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down IAM service")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}
	logger.Info("IAM service stopped gracefully")
}

// corsMiddleware adds CORS headers (consistent with gateway service).
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
