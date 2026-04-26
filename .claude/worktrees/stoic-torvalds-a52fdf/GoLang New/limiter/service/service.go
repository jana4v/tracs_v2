package service

import (
	"context"
	"fmt"
	"time"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository/sqlite"

	limiter "github.com/mainframe/tm-system/limiter/internal"
)

// LimiterConfig extends BaseConfig with limit monitor settings.
type LimiterConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	PollIntervalMs    int `mapstructure:"poll_interval_ms"`
}

// Run starts the limit monitor service and blocks until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	var cfg LimiterConfig
	if err := config.Load(configPath, &cfg); err != nil {
		return fmt.Errorf("limiter: config error: %w", err)
	}
	if cfg.PollIntervalMs == 0 {
		cfg.PollIntervalMs = 500
	}

	logger := logging.NewLogger(cfg.Service.Name, logging.ParseLevel(cfg.Service.LogLevel))

	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		return fmt.Errorf("limiter: failed to connect to Redis: %w", err)
	}
	defer rdb.Close()

	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		return fmt.Errorf("limiter: failed to open SQLite: %w", err)
	}
	defer sdb.Close()

	tmRepo := sqlite.NewTMMnemonicRepo(sdb, logger)
	loader := limiter.NewMnemonicLoader(tmRepo, logger)
	if err := loader.Load(ctx); err != nil {
		return fmt.Errorf("limiter: failed to load mnemonics: %w", err)
	}

	go func() {
		sub := rdb.Subscribe(ctx, models.MdbTmMnemonicsUpdated)
		defer sub.Close()

		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				if msg == nil {
					continue
				}
				logger.Info("received MDB_TM_MNEMONICS_UPDATED, reloading mnemonics")
				if err := loader.Reload(ctx); err != nil {
					logger.Error("failed to reload mnemonics", "error", err)
				}
			}
		}
	}()

	mon := limiter.NewMonitor(rdb, loader, logger)

	logger.Info("limit monitor started", "poll_interval_ms", cfg.PollIntervalMs)

	ticker := time.NewTicker(time.Duration(cfg.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("limit monitor stopped")
			return nil
		case <-ticker.C:
			mon.Check(ctx)
		}
	}
}
