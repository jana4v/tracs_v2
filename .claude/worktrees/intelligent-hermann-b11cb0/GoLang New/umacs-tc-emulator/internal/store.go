package internal

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// tcFilesStatusKey is the Redis hash key that umacs-tc polls for procedure status.
const tcFilesStatusKey = "TC_FILES_STATUS"

// EmulatorConfig controls the timing and behaviour of the simulated UMACS execution engine.
type EmulatorConfig struct {
	// QueuedDelay is the pause before moving a procedure from "queued" to "in-progress".
	QueuedDelay time.Duration
	// InProgressDuration is how long the "in-progress" phase lasts before completing.
	InProgressDuration time.Duration
	// SuccessRate is the percentage (0-100) of procedure executions that end in "success".
	// Any execution outside this rate ends in "failure".
	SuccessRate int
	// ValidateRequired controls whether createProcedure + validateProcedure must be
	// called before loadProcedure is accepted.
	ValidateRequired bool
}

// ProcRecord holds the emulated state for a single procedure.
type ProcRecord struct {
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	Validated bool      `json:"validated"`
	ExeStatus string    `json:"exe_status"`
	LoadedAt  time.Time `json:"loaded_at,omitempty"`
	Mode      int       `json:"mode"`
	Priority  int       `json:"priority"`
}

// ProcedureStore is a thread-safe in-memory registry of procedures and their
// simulated execution state. It optionally pushes status updates to Redis so
// that the umacs-tc service's polling loop (TC_FILES_STATUS) resolves correctly.
type ProcedureStore struct {
	mu     sync.RWMutex
	procs  map[string]*ProcRecord
	cfg    *EmulatorConfig
	rdb    *redis.Client // nil when Redis integration is disabled
	logger *slog.Logger
}

func NewProcedureStore(cfg *EmulatorConfig, rdb *redis.Client, logger *slog.Logger) *ProcedureStore {
	return &ProcedureStore{
		procs:  make(map[string]*ProcRecord),
		cfg:    cfg,
		rdb:    rdb,
		logger: logger.With("component", "store"),
	}
}

// Create registers a new procedure (createProcedure endpoint).
func (s *ProcedureStore) Create(procName, procedure string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.procs[procName] = &ProcRecord{
		Name:      procName,
		Content:   procedure,
		Validated: false,
		ExeStatus: "not-available",
	}
	s.logger.Info("procedure created", "proc_name", procName, "bytes", len(procedure))
	return nil
}

// Validate marks a stored procedure as validated (validateProcedure endpoint).
func (s *ProcedureStore) Validate(procName, procSrc, subsystem string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	proc, ok := s.procs[procName]
	if !ok {
		return fmt.Errorf("procedure '%s' not found — call createProcedure first", procName)
	}
	proc.Validated = true
	s.logger.Info("procedure validated",
		"proc_name", procName, "proc_src", procSrc, "subsystem", subsystem)
	return nil
}

// Load starts simulated execution of a procedure (loadProcedure endpoint).
// It returns immediately; the status transitions happen asynchronously.
func (s *ProcedureStore) Load(procName, procSrc string, procMode, procPriority int) error {
	s.mu.Lock()

	proc, ok := s.procs[procName]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("procedure '%s' not found — call createProcedure first", procName)
	}
	if s.cfg.ValidateRequired && !proc.Validated {
		s.mu.Unlock()
		return fmt.Errorf("procedure '%s' has not been validated — call validateProcedure first", procName)
	}

	proc.ExeStatus = "queued"
	proc.LoadedAt = time.Now()
	proc.Mode = procMode
	proc.Priority = procPriority
	s.mu.Unlock()

	s.logger.Info("procedure queued for execution",
		"proc_name", procName, "proc_mode", procMode, "proc_priority", procPriority)

	go s.simulateExecution(procName)
	return nil
}

// simulateExecution drives the state machine: queued → in-progress → success|failure.
// If Redis is configured it also updates TC_FILES_STATUS so umacs-tc's polling loop
// is satisfied without needing getExeStatus polling.
func (s *ProcedureStore) simulateExecution(procName string) {
	// ── queued → in-progress ─────────────────────────────────────────────────
	time.Sleep(s.cfg.QueuedDelay)

	s.mu.Lock()
	proc, ok := s.procs[procName]
	if !ok || proc.ExeStatus != "queued" {
		s.mu.Unlock()
		return
	}
	proc.ExeStatus = "in-progress"
	s.mu.Unlock()

	s.logger.Info("procedure in-progress", "proc_name", procName)
	s.redisSet(procName, "in-progress")

	// ── in-progress → success|failure ────────────────────────────────────────
	time.Sleep(s.cfg.InProgressDuration)

	s.mu.Lock()
	proc, ok = s.procs[procName]
	if !ok || proc.ExeStatus != "in-progress" {
		s.mu.Unlock()
		return
	}

	outcome := "success"
	if s.cfg.SuccessRate < 100 && rand.Intn(100) >= s.cfg.SuccessRate {
		outcome = "failure"
	}
	proc.ExeStatus = outcome
	s.mu.Unlock()

	s.logger.Info("procedure execution complete", "proc_name", procName, "outcome", outcome)
	s.redisSet(procName, outcome)
}

// redisSet writes the status to TC_FILES_STATUS in Redis when Redis is available.
func (s *ProcedureStore) redisSet(procName, status string) {
	if s.rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.rdb.HSet(ctx, tcFilesStatusKey, procName, status).Err(); err != nil {
		s.logger.Warn("failed to update Redis TC_FILES_STATUS",
			"proc_name", procName, "status", status, "error", err)
	} else {
		s.logger.Debug("Redis TC_FILES_STATUS updated",
			"proc_name", procName, "status", status)
	}
}

// GetExeStatus returns the current execution status of a procedure (getExeStatus endpoint).
func (s *ProcedureStore) GetExeStatus(procName string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	proc, ok := s.procs[procName]
	if !ok {
		return "not-available", nil
	}
	return proc.ExeStatus, nil
}

// All returns a snapshot of all stored procedure records (admin/debug use).
func (s *ProcedureStore) All() map[string]ProcRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]ProcRecord, len(s.procs))
	for k, v := range s.procs {
		out[k] = *v
	}
	return out
}
