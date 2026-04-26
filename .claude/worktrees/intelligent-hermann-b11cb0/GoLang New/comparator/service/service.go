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

	comp "github.com/mainframe/tm-system/comparator/internal"
)

// ComparatorConfig extends BaseConfig with comparator settings.
type ComparatorConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	ComparePairs      []ComparePair `mapstructure:"compare_pairs"`
	PollIntervalMs    int           `mapstructure:"poll_interval_ms"`
}

// ComparePair defines a pair of chains to compare.
type ComparePair struct {
	Chain1 string `mapstructure:"chain1"`
	Chain2 string `mapstructure:"chain2"`
}

// Run starts the chain comparator service and blocks until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	var cfg ComparatorConfig
	if err := config.Load(configPath, &cfg); err != nil {
		return fmt.Errorf("comparator: config error: %w", err)
	}
	if cfg.PollIntervalMs == 0 {
		cfg.PollIntervalMs = 500
	}

	logger := logging.NewLogger(cfg.Service.Name, logging.ParseLevel(cfg.Service.LogLevel))

	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		return fmt.Errorf("comparator: failed to connect to Redis: %w", err)
	}
	defer rdb.Close()

	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		return fmt.Errorf("comparator: failed to open SQLite: %w", err)
	}
	defer sdb.Close()

	tmRepo := sqlite.NewTMMnemonicRepo(sdb, logger)
	loader := comp.NewMnemonicLoader(tmRepo, logger)
	if err := loader.Load(ctx); err != nil {
		return fmt.Errorf("comparator: failed to load mnemonics: %w", err)
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

	pairs := make([]comp.ChainPair, len(cfg.ComparePairs))
	for i, p := range cfg.ComparePairs {
		pairs[i] = comp.ChainPair{Chain1: p.Chain1, Chain2: p.Chain2}
	}

	comparator := comp.NewComparator(rdb, loader, pairs, logger)

	logger.Info("chain comparator started",
		"pairs", len(pairs),
		"poll_interval_ms", cfg.PollIntervalMs,
	)

	ticker := time.NewTicker(time.Duration(cfg.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("chain comparator stopped")
			return nil
		case <-ticker.C:
			comparator.Compare(ctx)
		}
	}
}
