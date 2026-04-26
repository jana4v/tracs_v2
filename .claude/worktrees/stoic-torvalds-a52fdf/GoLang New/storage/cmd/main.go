package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository/sqlite"

	store "github.com/mainframe/tm-system/storage/internal"
)

// StorageConfig extends BaseConfig with storage-specific settings.
type StorageConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	Influx            config.InfluxConfig `mapstructure:"influx"`
	PollIntervalMs    int                 `mapstructure:"poll_interval_ms"`
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	var cfg StorageConfig
	if err := config.Load(*configPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if cfg.PollIntervalMs == 0 {
		cfg.PollIntervalMs = 500
	}

	logger := logging.NewLogger(cfg.Service.Name, logging.ParseLevel(cfg.Service.LogLevel))

	// Context with graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Connect to Redis
	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Open SQLite
	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		logger.Error("failed to open SQLite", "error", err)
		os.Exit(1)
	}
	defer sdb.Close()

	tmRepo := sqlite.NewTMMnemonicRepo(sdb, logger)

	// Load mnemonics with enable_storage=true
	loader := store.NewMnemonicLoader(tmRepo, logger)
	if err := loader.Load(ctx); err != nil {
		logger.Error("failed to load mnemonics", "error", err)
		os.Exit(1)
	}

	// Create storage components
	cache := store.NewCache()
	rules := store.NewRuleEngine(cache, rdb, logger)

	var influxClient *influxdb3.Client
	var writer *store.Writer
	defer func() {
		if influxClient != nil {
			influxClient.Close()
		}
	}()

	if rules.IsGlobalStorageEnabled(ctx) {
		influxClient, err = clients.NewInfluxClient(ctx, cfg.Influx.URL, cfg.Influx.Token, cfg.Influx.Database, logger)
		if err != nil {
			logger.Error("initial InfluxDB connection failed; storage writes will remain disabled until config enables successful reconnect", "error", err)
		} else {
			writer = store.NewWriter(influxClient, logger)
		}
	} else {
		logger.Info("InfluxDB logging disabled by software config; skipping InfluxDB connection")
	}

	// Subscribe to MDB_TM_MNEMONICS_UPDATED for event-driven reload
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

	logger.Info("storage service started", "poll_interval_ms", cfg.PollIntervalMs)

	ticker := time.NewTicker(time.Duration(cfg.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("storage service stopped")
			return
		case <-ticker.C:
			enabled := rules.IsGlobalStorageEnabled(ctx)
			if !enabled {
				if influxClient != nil {
					logger.Info("InfluxDB logging disabled by software config; closing InfluxDB connection")
					influxClient.Close()
					influxClient = nil
					writer = nil
				}
				continue
			}

			if influxClient == nil {
				influxClient, err = clients.NewInfluxClient(ctx, cfg.Influx.URL, cfg.Influx.Token, cfg.Influx.Database, logger)
				if err != nil {
					logger.Error("failed to connect to InfluxDB; skipping this cycle", "error", err)
					continue
				}
				writer = store.NewWriter(influxClient, logger)
			}

			// Read TM_MAP
			tmData, err := rdb.HGetAll(ctx, models.TMMap).Result()
			if err != nil {
				logger.Error("failed to read TM_MAP", "error", err)
				continue
			}

			if len(tmData) == 0 {
				continue
			}

			mnemonics := loader.Get()
			now := time.Now().UTC()

			for _, mnem := range mnemonics {
				id := string(mnem.ID)
				value, exists := tmData[id]
				if !exists {
					continue
				}

				if rules.ShouldStore(mnem, value, now) {
					writer.Write(ctx, mnem, value, now)
					cache.Update(id, value, now, false)
				}
			}
		}
	}
}
