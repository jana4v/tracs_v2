package service

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/logging"
	"github.com/mainframe/tm-system/internal/repository/sqlite"

	ingest "github.com/mainframe/tm-system/ingest/internal"
)

// IngestConfig extends BaseConfig with chain definitions, WebSocket settings,
// NATS config, and the simulation API port.
type IngestConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	WebSocket         config.WebSocketConfig `mapstructure:"websocket"`
	Chains            []config.ChainConfig   `mapstructure:"chains"`
	NATS              config.NATSConfig      `mapstructure:"nats"`
	SimPort           int                    `mapstructure:"sim_port"` // default 8082
}

// Run starts the ingest service and blocks until ctx is cancelled.
func Run(ctx context.Context, configPath string) error {
	var cfg IngestConfig
	if err := config.Load(configPath, &cfg); err != nil {
		return fmt.Errorf("ingest: config error: %w", err)
	}

	if cfg.SimPort == 0 {
		cfg.SimPort = 8082
	}

	logger := logging.NewLogger(cfg.Service.Name, logging.ParseLevel(cfg.Service.LogLevel))

	rdb, err := clients.NewRedisClient(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		return fmt.Errorf("ingest: failed to connect to Redis: %w", err)
	}
	defer rdb.Close()

	sdb, err := clients.NewSQLiteDB(ctx, cfg.SQLite, logger)
	if err != nil {
		return fmt.Errorf("ingest: failed to open SQLite: %w", err)
	}
	defer sdb.Close()

	// --- StreamStore ---
	store := ingest.NewStreamStore()
	store.GetOrCreate(ingest.StreamMeta{ID: "DTM", ChainType: "programmatic", ChainName: "DTM"})
	store.GetOrCreate(ingest.StreamMeta{ID: "UDTM", ChainType: "programmatic", ChainName: "UDTM"})

	// --- Chain names for status publishing ---
	chainNames := make([]string, 0, len(cfg.Chains))
	for _, cc := range cfg.Chains {
		chainNames = append(chainNames, cc.Name)
	}

	// --- NATS (optional) ---
	var natsClient *clients.NATSClient
	if nc, err := clients.NewNATSClient(cfg.NATS.URL, cfg.NATS.Name, logger); err != nil {
		logger.Warn("NATS unavailable — publishing disabled", "error", err)
	} else {
		natsClient = nc
		defer natsClient.Close()
	}

	// --- StreamDispatcher (replaces NATSPublisher) ---
	dispatcher := ingest.NewStreamDispatcher(store, rdb, natsClient, cfg.NATS, logger, chainNames)
	go dispatcher.Run(ctx)

	// --- SimManager ---
	simManager := ingest.NewSimManager(store, logger)

	// --- Repository layer for simulator config sync ---
	tmRepo := sqlite.NewTMMnemonicRepo(sdb, logger)
	dtmRepo := sqlite.NewDTMRepo(sdb, logger)
	udtmRepo := sqlite.NewUDTMRepo(sdb, logger)

	// --- Redis-driven simulator config sync (GUI chain-simulator state) ---
	simConfigSync := ingest.NewSimConfigSync(rdb, tmRepo, dtmRepo, udtmRepo, store, simManager, logger)
	go simConfigSync.Run(ctx)

	// --- Sim API ---
	simMux := ingest.NewSimMux(store, simManager, ctx, logger)
	simAddr := fmt.Sprintf(":%d", cfg.SimPort)
	simSrv := &http.Server{
		Addr:         simAddr,
		Handler:      simMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		logger.Info("sim API started", "addr", simAddr)
		if err := simSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("sim API server error", "error", err)
		}
	}()

	simSync := ingest.NewSimSync(rdb, logger)
	go simSync.Run(ctx, chainNames, 1*time.Second)

	if len(cfg.Chains) == 0 {
		return fmt.Errorf("ingest: no chains configured in config file")
	}

	var wg sync.WaitGroup
	for _, chain := range cfg.Chains {
		wg.Add(1)
		go func(cc config.ChainConfig) {
			defer wg.Done()
			chainLogger := logger.With("chain", cc.Name, "type", cc.Type)
			sub := ingest.NewChainSubscriber(cc, rdb, store, chainLogger)
			if err := sub.Run(ctx); err != nil && ctx.Err() == nil {
				chainLogger.Error("chain subscriber exited with error", "error", err)
			}
		}(chain)
	}

	logger.Info("tm-ingest started", "chains", len(cfg.Chains), "sim_port", cfg.SimPort)

	<-ctx.Done()
	logger.Info("shutting down tm-ingest...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := simSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("sim server shutdown error", "error", err)
	}
	wg.Wait()
	logger.Info("tm-ingest stopped")
	return nil
}
