package main

// umacs-tc-emulator — a zero-hardware stand-in for the UMACS TC REST interface.
//
// It listens on port 21003 (the configured UMACS TC target port) and implements:
//
//	POST /createProcedure    — store procedure content
//	POST /validateProcedure  — mark procedure as validated
//	POST /loadProcedure      — queue & execute (async simulation)
//	POST /getExeStatus       — query current execution status
//	GET  /admin/procedures   — debug snapshot of all procedures
//	GET  /health             — liveness probe
//
// # Redis integration (optional)
//
// When --redis-addr is provided, the emulator writes status transitions to
// the TC_FILES_STATUS Redis hash key — the same key that umacs-tc's
// triggerFileWaitForExecutionComplete() polls. This lets the full
//
//	Julia procedure → umacs-tc → [emulator] → Redis poll
//
// chain work end-to-end without real UMACS hardware.
//
// # Typical usage
//
//	# Standalone (no Redis required)
//	./umacs-tc-emulator
//
//	# Full integration with Redis
//	./umacs-tc-emulator --redis-addr localhost:6379
//
//	# Faster simulation (500 ms total)
//	./umacs-tc-emulator --queued-delay 100 --inprogress-duration 400
//
//	# Simulate 20% failure rate
//	./umacs-tc-emulator --success-rate 80
//
//	# Skip validation requirement (accept loadProcedure without prior validate)
//	./umacs-tc-emulator --no-validate-required

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mainframe/umacs-tc-emulator/internal"
	"github.com/redis/go-redis/v9"
)

func main() {
	// ── CLI flags ────────────────────────────────────────────────────────────
	host := flag.String("host", "0.0.0.0", "Emulator listen host")
	port := flag.Int("port", 21003, "Emulator listen port (matches configured UMACS TC target port)")

	queuedDelayMs := flag.Int("queued-delay", 500,
		"Milliseconds before queued → in-progress transition")
	inProgressMs := flag.Int("inprogress-duration", 4000,
		"Milliseconds the in-progress phase lasts before completing")
	successRate := flag.Int("success-rate", 100,
		"Percentage (0-100) of executions that succeed; remainder result in failure")
	noValidateRequired := flag.Bool("no-validate-required", false,
		"Accept loadProcedure without prior validateProcedure call")

	redisAddr := flag.String("redis-addr", "",
		"Redis address (e.g. localhost:6379). When set, status updates are written "+
			"to TC_FILES_STATUS so umacs-tc's polling loop is satisfied.")
	redisPassword := flag.String("redis-password", "", "Redis password (optional)")
	redisDB := flag.Int("redis-db", 0, "Redis database index")

	flag.Parse()

	// ── Logger ───────────────────────────────────────────────────────────────
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// ── Redis (optional) ─────────────────────────────────────────────────────
	var rdb *redis.Client
	if *redisAddr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     *redisAddr,
			Password: *redisPassword,
			DB:       *redisDB,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			logger.Error("Redis connection failed — continuing without Redis integration",
				"addr", *redisAddr, "error", err)
			rdb = nil
		} else {
			logger.Info("Redis connected — TC_FILES_STATUS updates enabled", "addr", *redisAddr)
			defer rdb.Close()
		}
	} else {
		logger.Info("Redis not configured — TC_FILES_STATUS updates disabled; " +
			"use --redis-addr to enable full umacs-tc integration")
	}

	// ── Emulator config ───────────────────────────────────────────────────────
	cfg := &internal.EmulatorConfig{
		QueuedDelay:        time.Duration(*queuedDelayMs) * time.Millisecond,
		InProgressDuration: time.Duration(*inProgressMs) * time.Millisecond,
		SuccessRate:        *successRate,
		ValidateRequired:   !*noValidateRequired,
	}

	logger.Info("emulator config",
		"queued_delay", cfg.QueuedDelay,
		"inprogress_duration", cfg.InProgressDuration,
		"success_rate", cfg.SuccessRate,
		"validate_required", cfg.ValidateRequired,
	)

	// ── Wire up ──────────────────────────────────────────────────────────────
	store := internal.NewProcedureStore(cfg, rdb, logger)
	handler := internal.NewHandler(store, logger)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Start server ─────────────────────────────────────────────────────────
	go func() {
		logger.Info("UMACS TC Emulator listening",
			"addr", addr,
			"endpoints", []string{
				"POST /createProcedure",
				"POST /validateProcedure",
				"POST /loadProcedure",
				"POST /getExeStatus",
				"GET  /admin/procedures",
				"GET  /health",
			},
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down emulator...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("emulator stopped")
}
