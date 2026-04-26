package ingest

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mainframe/tm-system/internal/models"
)

type SimSync struct {
	rdb    *redis.Client
	logger *slog.Logger
}

func NewSimSync(rdb *redis.Client, logger *slog.Logger) *SimSync {
	return &SimSync{
		rdb:    rdb,
		logger: logger,
	}
}

func (ss *SimSync) IsSimulatorEnabled(ctx context.Context) bool {
	enable, err := ss.rdb.HGet(ctx, models.TMSimulatorCfgMap, models.SimCfgEnable).Result()
	if err != nil || enable != "1" {
		return false
	}
	return true
}

func (ss *SimSync) IsWebSocketDataFlowing(ctx context.Context, chainNames []string) bool {
	for _, name := range chainNames {
		status, err := ss.rdb.Get(ctx, models.HeartbeatStatusKey(name)).Result()
		if err == nil && status == models.StatusOK {
			return true
		}
	}
	return false
}

func (ss *SimSync) SyncSimulatedToTMMap(ctx context.Context) error {
	simData, err := ss.rdb.HGetAll(ctx, models.SimulatedTMMap).Result()
	if err != nil {
		return err
	}

	if len(simData) == 0 {
		return nil
	}

	pipe := ss.rdb.Pipeline()
	for param, value := range simData {
		pipe.HSet(ctx, models.TMMap, param, value)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (ss *SimSync) StopSimulator(ctx context.Context) error {
	ss.logger.Info("websocket data resumed, stopping simulator")
	ss.rdb.Publish(ctx, models.TMSimulatorCtrlChannel, "stop")
	return ss.rdb.HSet(ctx, models.TMSimulatorCfgMap, models.SimCfgEnable, "0").Err()
}

func (ss *SimSync) Run(ctx context.Context, chainNames []string, checkInterval time.Duration) {
	ss.logger.Info("simulation sync loop started", "check_interval", checkInterval)
	wasSimulating := false
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ss.logger.Info("simulation sync loop stopped")
			return
		case <-ticker.C:
			simEnabled := ss.IsSimulatorEnabled(ctx)
			dataFlowing := ss.IsWebSocketDataFlowing(ctx, chainNames)

			if simEnabled && !dataFlowing {
				if err := ss.SyncSimulatedToTMMap(ctx); err != nil {
					ss.logger.Error("failed to sync simulated values to TM_MAP", "error", err)
				} else {
					ss.logger.Debug("synced simulated values to TM_MAP")
				}
				wasSimulating = true
			} else if wasSimulating && dataFlowing {
				if err := ss.StopSimulator(ctx); err != nil {
					ss.logger.Error("failed to stop simulator", "error", err)
				}
				wasSimulating = false
			}
		}
	}
}
