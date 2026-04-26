package internal

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
)

// Simulator implements the TM Simulator Service (SRS Section 3).
type Simulator struct {
	rdb       *redis.Client
	tmStore   repository.TMMnemonicStore
	logger    *slog.Logger
	mnemonics []models.TmMnemonic
	stopCh    chan struct{}
	runCh     chan struct{}
	running   bool
}

// NewSimulator creates a new Simulator instance.
func NewSimulator(rdb *redis.Client, tmStore repository.TMMnemonicStore, logger *slog.Logger) *Simulator {
	return &Simulator{
		rdb:     rdb,
		tmStore: tmStore,
		logger:  logger,
		stopCh:  make(chan struct{}),
		runCh:   make(chan struct{}, 1),
	}
}

// LoadMnemonics loads all mnemonic definitions from the TMMnemonicStore.
func (s *Simulator) LoadMnemonics(ctx context.Context) error {
	all, err := s.tmStore.FindAll(ctx)
	if err != nil {
		return err
	}

	s.mnemonics = nil
	for _, m := range all {
		if m.CdbMnemonic == "" {
			s.logger.Warn("skip mnemonic with empty cdbMnemonic", "id", m.ID)
			continue
		}
		s.mnemonics = append(s.mnemonics, m)
	}
	s.logger.Info("loaded mnemonics from SQLite", "count", len(s.mnemonics))
	return nil
}

// Run starts the simulator loop. It waits for start command via Redis pub/sub.
// SRS Section 3.3: generates values per MODE, writes to SIMULATED_TM_MAP, publishes heartbeat.
func (s *Simulator) Run(ctx context.Context) {
	s.logger.Info("simulator loop started, waiting for start command")

	pubsub := s.rdb.Subscribe(ctx, models.TMSimulatorCtrlChannel)
	defer pubsub.Close()

	enableVal, err := s.rdb.HGet(ctx, models.TMSimulatorCfgMap, models.SimCfgEnable).Result()
	if err != nil && err != redis.Nil {
		s.logger.Warn("failed to read simulator enable flag", "error", err)
	}
	if strings.EqualFold(strings.TrimSpace(enableVal), "1") || strings.EqualFold(strings.TrimSpace(enableVal), "true") {
		s.running = true
		go s.runSimulation(ctx)
		s.logger.Info("simulator auto-started from config", "enable", enableVal)
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("simulator loop stopped")
			return
		case <-s.stopCh:
			s.logger.Info("simulator loop stopped")
			return
		case msg := <-pubsub.Channel():
			if msg == nil {
				continue
			}
			s.logger.Info("received control command", "command", msg.Payload)

			if msg.Payload == "stop" {
				s.running = false
				s.logger.Info("simulator stopped")
				continue
			}

			if msg.Payload == "start" && !s.running {
				s.running = true
				go s.runSimulation(ctx)
			}
		}
	}
}

func (s *Simulator) runSimulation(ctx context.Context) {
	s.logger.Info("simulation started")
	prevMode := ""

	for {
		if !s.running {
			s.logger.Info("simulation loop ended")
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			s.logger.Info("simulation stopped")
			return
		default:
		}

		mode, err := s.GetMode(ctx)
		if err != nil {
			s.logger.Error("failed to get mode", "error", err)
			time.Sleep(time.Second)
			continue
		}

		if mode == "" {
			mode = models.SimModeFixed
		}

		if mode == models.SimModeFixed {
			if prevMode != models.SimModeFixed {
				s.logger.Info("FIXED mode: writing initial values")
				pipe := s.rdb.Pipeline()
				for _, m := range s.mnemonics {
					pipe.HSet(ctx, models.SimulatedTMMap, m.CdbMnemonic, InitialValue(m))
				}
				if _, err := pipe.Exec(ctx); err != nil {
					s.logger.Error("failed to write fixed values", "error", err)
				}
				s.rdb.Publish(ctx, models.TMSimulatorChannel, "heartbeat")
			}
			prevMode = mode
			time.Sleep(time.Duration(1000) * time.Millisecond)
			continue
		}

		cfg, err := s.rdb.HGetAll(ctx, models.TMSimulatorCfgMap).Result()
		if err != nil {
			s.logger.Error("failed to read simulator config", "error", err)
			time.Sleep(time.Second)
			continue
		}

		sampleDelayMs, _ := strconv.Atoi(cfg[models.SimCfgSampleDelay])
		if sampleDelayMs <= 0 {
			sampleDelayMs = 1000
		}

		pipe := s.rdb.Pipeline()
		for _, m := range s.mnemonics {
			val := GenerateValue(m, mode)
			pipe.HSet(ctx, models.SimulatedTMMap, m.CdbMnemonic, val)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			s.logger.Error("failed to write simulated values", "error", err)
		}

		s.rdb.Publish(ctx, models.TMSimulatorChannel, "heartbeat")
		prevMode = mode

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(sampleDelayMs) * time.Millisecond):
		}
	}
}

// Reset resets all mnemonic values to range[0] (SRS 3.4.2).
func (s *Simulator) Reset(ctx context.Context) error {
	pipe := s.rdb.Pipeline()
	for _, m := range s.mnemonics {
		pipe.HSet(ctx, models.SimulatedTMMap, m.CdbMnemonic, InitialValue(m))
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	s.logger.Info("simulator reset to initial values", "count", len(s.mnemonics))
	return nil
}

// UpdateValues updates specific mnemonic values in SIMULATED_TM_MAP (SRS 3.4.1).
func (s *Simulator) UpdateValues(ctx context.Context, updates map[string]string) error {
	pipe := s.rdb.Pipeline()
	for mnemonic, value := range updates {
		pipe.HSet(ctx, models.SimulatedTMMap, mnemonic, value)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		s.logger.Error("pipeline exec failed", "error", err)
	}
	return err
}

// GetStatus returns current simulator config and mnemonic count (SRS 3.4.3).
func (s *Simulator) GetStatus(ctx context.Context) (map[string]interface{}, error) {
	cfg, err := s.rdb.HGetAll(ctx, models.TMSimulatorCfgMap).Result()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"config":         cfg,
		"mnemonic_count": len(s.mnemonics),
	}, nil
}

// MnemonicCount returns the number of loaded mnemonics.
func (s *Simulator) MnemonicCount() int {
	return len(s.mnemonics)
}

// GetMnemonics returns the loaded mnemonic definitions.
func (s *Simulator) GetMnemonics() []models.TmMnemonic {
	return s.mnemonics
}

func (s *Simulator) GetSubsystems() []string {
	if len(s.mnemonics) == 0 {
		return []string{}
	}
	unique := make(map[string]struct{})
	for _, m := range s.mnemonics {
		name := strings.TrimSpace(m.Subsystem)
		if name == "" {
			continue
		}
		unique[name] = struct{}{}
	}
	if len(unique) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(unique))
	for name := range unique {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (s *Simulator) EnsureConfig(ctx context.Context) error {
	cfg, err := s.rdb.HGetAll(ctx, models.TMSimulatorCfgMap).Result()
	if err != nil {
		return err
	}
	updates := map[string]interface{}{}
	if cfg[models.SimCfgEnable] == "" {
		updates[models.SimCfgEnable] = "1"
	}
	if cfg[models.SimCfgMode] == "" {
		updates[models.SimCfgMode] = models.SimModeFixed
	}
	if cfg[models.SimCfgSampleDelay] == "" {
		updates[models.SimCfgSampleDelay] = "1000"
	}
	if len(updates) == 0 {
		return nil
	}
	return s.rdb.HSet(ctx, models.TMSimulatorCfgMap, updates).Err()
}

func (s *Simulator) GetMode(ctx context.Context) (string, error) {
	mode, err := s.rdb.HGet(ctx, models.TMSimulatorCfgMap, models.SimCfgMode).Result()
	if err != nil {
		if err == redis.Nil {
			return models.SimModeFixed, nil
		}
		return "", err
	}
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode == "" {
		return models.SimModeFixed, nil
	}
	if mode != models.SimModeFixed && mode != models.SimModeRandom {
		return models.SimModeFixed, nil
	}
	return mode, nil
}

func (s *Simulator) SetMode(ctx context.Context, mode string) error {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode == "" {
		mode = models.SimModeFixed
	}
	return s.rdb.HSet(ctx, models.TMSimulatorCfgMap, models.SimCfgMode, mode).Err()
}

func (s *Simulator) Start(ctx context.Context) error {
	s.rdb.Publish(ctx, models.TMSimulatorCtrlChannel, "start")
	return s.rdb.HSet(ctx, models.TMSimulatorCfgMap, models.SimCfgEnable, "1").Err()
}

func (s *Simulator) Stop(ctx context.Context) error {
	s.rdb.Publish(ctx, models.TMSimulatorCtrlChannel, "stop")
	_ = s.rdb.HSet(ctx, models.TMSimulatorCfgMap, models.SimCfgEnable, "0").Err()
	return s.rdb.Del(ctx, models.SimulatedTMMap).Err()
}

type SimulatedValue struct {
	Mnemonic string `json:"mnemonic"`
	Value    string `json:"value"`
}

func (s *Simulator) GetValues(ctx context.Context) ([]SimulatedValue, error) {
	data, err := s.rdb.HGetAll(ctx, models.SimulatedTMMap).Result()
	if err != nil {
		return nil, err
	}
	result := make([]SimulatedValue, 0, len(data))
	for k, v := range data {
		result = append(result, SimulatedValue{
			Mnemonic: k,
			Value:    v,
		})
	}
	return result, nil
}

func (s *Simulator) GetValuesBySubsystems(ctx context.Context, subsystems []string) ([]SimulatedValue, error) {
	if len(subsystems) == 0 {
		return []SimulatedValue{}, nil
	}
	allowed := make(map[string]struct{}, len(subsystems))
	for _, raw := range subsystems {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		allowed[name] = struct{}{}
	}
	if len(allowed) == 0 {
		return []SimulatedValue{}, nil
	}
	keys := make([]string, 0)
	for _, m := range s.mnemonics {
		if m.CdbMnemonic == "" {
			continue
		}
		if _, ok := allowed[m.Subsystem]; ok {
			keys = append(keys, m.CdbMnemonic)
		}
	}
	if len(keys) == 0 {
		return []SimulatedValue{}, nil
	}
	values, err := s.rdb.HMGet(ctx, models.SimulatedTMMap, keys...).Result()
	if err != nil {
		return nil, err
	}
	result := make([]SimulatedValue, 0, len(keys))
	for i, v := range values {
		if v == nil {
			continue
		}
		result = append(result, SimulatedValue{
			Mnemonic: keys[i],
			Value:    fmt.Sprintf("%v", v),
		})
	}
	return result, nil
}
