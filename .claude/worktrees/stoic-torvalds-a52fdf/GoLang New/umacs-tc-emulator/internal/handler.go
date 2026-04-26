package internal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// ─── Wire types ──────────────────────────────────────────────────────────────

// Response is the UMACS TC REST API response envelope.
// exe_status and error_msg are always present (never omitted) to match the
// documented interface format exactly.
type Response struct {
	Ack       bool   `json:"ack"`
	ErrorMsg  string `json:"error_msg"`
	ExeStatus string `json:"exe_status"`
}

// createRequest maps to the createProcedure endpoint body.
type createRequest struct {
	ProcName  string `json:"proc_name"`
	Procedure string `json:"procedure"`
}

// validateRequest maps to the validateProcedure endpoint body.
// Both proc_src (spec) and proc_source (umacs-tc handler compat) are accepted.
type validateRequest struct {
	ProcName   string `json:"proc_name"`
	ProcSrc    string `json:"proc_src,omitempty"`
	ProcSource string `json:"proc_source,omitempty"` // alias used by umacs-tc handler
	SubSystem  string `json:"subsystem"`
}

// loadRequest maps to both loadProcedure and getExeStatus endpoint bodies.
// proc_mode and proc_priority are represented as json.RawMessage so they can
// be either JSON integers or quoted strings (both forms appear in the wild).
type loadRequest struct {
	Action       string          `json:"action"`
	ProcName     string          `json:"proc_name"`
	ProcSrc      string          `json:"proc_src,omitempty"`
	ProcMode     json.RawMessage `json:"proc_mode,omitempty"`
	ProcPriority json.RawMessage `json:"proc_priority,omitempty"`
}

// ─── Handler ─────────────────────────────────────────────────────────────────

type Handler struct {
	store  *ProcedureStore
	logger *slog.Logger
}

func NewHandler(store *ProcedureStore, logger *slog.Logger) *Handler {
	return &Handler{
		store:  store,
		logger: logger.With("component", "handler"),
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// UMACS TC interface endpoints (port 21003)
	mux.HandleFunc("/api/go/v1/createProcedure", h.createProcedure)
	mux.HandleFunc("/api/go/v1/validateProcedure", h.validateProcedure)
	mux.HandleFunc("/api/go/v1/loadProcedure", h.loadProcedure)
	mux.HandleFunc("/api/go/v1/getExeStatus", h.getExeStatus)

	// Debug/admin — not part of UMACS spec
	mux.HandleFunc("/api/go/v1/admin/procedures", h.adminProcedures)
	mux.HandleFunc("/api/go/v1/health", h.health)
}

// ─── Endpoint handlers ────────────────────────────────────────────────────────

// createProcedure stores the procedure content on the emulated UMACS server.
//
// Request:  { "proc_name": "test1.tst", "procedure": "<full procedure text>" }
// Response: { "ack": true, "error_msg": "procedure created successfully", "exe_status": "" }
func (h *Handler) createProcedure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("createProcedure decode error", "error", err)
		writeJSON(w, Response{Ack: false, ErrorMsg: fmt.Sprintf("invalid JSON: %s", err)})
		return
	}

	h.logger.Info("createProcedure", "proc_name", req.ProcName, "bytes", len(req.Procedure))

	if req.ProcName == "" {
		writeJSON(w, Response{Ack: false, ErrorMsg: "proc_name is required"})
		return
	}
	if req.Procedure == "" {
		writeJSON(w, Response{Ack: false, ErrorMsg: "procedure content is required"})
		return
	}

	if err := h.store.Create(req.ProcName, req.Procedure); err != nil {
		writeJSON(w, Response{Ack: false, ErrorMsg: err.Error()})
		return
	}

	writeJSON(w, Response{Ack: true, ErrorMsg: "procedure created successfully"})
}

// validateProcedure validates a previously created procedure.
//
// Request:  { "proc_name": "test1.tst", "proc_src": "integration-test", "subsystem": "PAYLOAD" }
// Response: { "ack": true, "error_msg": "procedure validated successfully", "exe_status": "" }
func (h *Handler) validateProcedure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("validateProcedure decode error", "error", err)
		writeJSON(w, Response{Ack: false, ErrorMsg: fmt.Sprintf("invalid JSON: %s", err)})
		return
	}

	// Accept proc_src (spec) or proc_source (umacs-tc compat)
	procSrc := req.ProcSrc
	if procSrc == "" {
		procSrc = req.ProcSource
	}

	h.logger.Info("validateProcedure",
		"proc_name", req.ProcName, "proc_src", procSrc, "subsystem", req.SubSystem)

	if req.ProcName == "" {
		writeJSON(w, Response{Ack: false, ErrorMsg: "proc_name is required"})
		return
	}

	if err := h.store.Validate(req.ProcName, procSrc, req.SubSystem); err != nil {
		writeJSON(w, Response{Ack: false, ErrorMsg: err.Error()})
		return
	}

	writeJSON(w, Response{Ack: true, ErrorMsg: "procedure validated successfully"})
}

// loadProcedure queues a validated procedure for execution and immediately
// returns ack=true. Actual execution status is tracked asynchronously and can
// be queried via getExeStatus. If Redis is configured, the emulator also
// writes updates to TC_FILES_STATUS so umacs-tc's internal polling loop works.
//
// Request:  { "action": "execute", "proc_name": "test1.tst",
//
//	"proc_src": "integration-test", "proc_mode": 1, "proc_priority": 1 }
//
// Response: { "ack": true, "error_msg": "procedure loaded successfully", "exe_status": "" }
func (h *Handler) loadProcedure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("loadProcedure decode error", "error", err)
		writeJSON(w, Response{Ack: false, ErrorMsg: fmt.Sprintf("invalid JSON: %s", err)})
		return
	}

	if req.ProcName == "" {
		writeJSON(w, Response{Ack: false, ErrorMsg: "proc_name is required"})
		return
	}

	procMode := parseIntOrString(req.ProcMode)
	procPriority := parseIntOrString(req.ProcPriority)

	h.logger.Info("loadProcedure",
		"action", req.Action,
		"proc_name", req.ProcName,
		"proc_src", req.ProcSrc,
		"proc_mode", procMode,
		"proc_priority", procPriority)

	if err := h.store.Load(req.ProcName, req.ProcSrc, procMode, procPriority); err != nil {
		writeJSON(w, Response{Ack: false, ErrorMsg: err.Error()})
		return
	}

	writeJSON(w, Response{Ack: true, ErrorMsg: "procedure loaded successfully"})
}

// getExeStatus returns the current execution status of a procedure.
//
// Request:  { "action": "exestatus", "proc_name": "test1.tst",
//
//	"proc_src": "integration-test", "proc_mode": 0, "proc_priority": 0 }
//
// Response: { "ack": true, "error_msg": "", "exe_status": "in-progress" }
//
// Possible exe_status values: queued | in-progress | success | failure |
// aborted | suspended | not-available
func (h *Handler) getExeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("getExeStatus decode error", "error", err)
		writeJSON(w, Response{Ack: false, ErrorMsg: fmt.Sprintf("invalid JSON: %s", err)})
		return
	}

	if req.ProcName == "" {
		writeJSON(w, Response{Ack: false, ErrorMsg: "proc_name is required"})
		return
	}

	status, err := h.store.GetExeStatus(req.ProcName)
	if err != nil {
		writeJSON(w, Response{Ack: false, ErrorMsg: err.Error()})
		return
	}

	h.logger.Debug("getExeStatus", "proc_name", req.ProcName, "exe_status", status)
	writeJSON(w, Response{Ack: true, ExeStatus: status})
}

// ─── Admin endpoints (not part of UMACS spec) ─────────────────────────────────

// adminProcedures returns a JSON snapshot of all stored procedures for debugging.
func (h *Handler) adminProcedures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.store.All())
}

// health is a simple liveness probe.
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok"}`)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// parseIntOrString coerces a json.RawMessage that might be a JSON integer
// (e.g. 1) or a JSON string (e.g. "1", "auto", "normal") into an int.
// Named values follow the UMACS annexure interpretation table.
func parseIntOrString(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}

	// Try bare integer first.
	var i int
	if err := json.Unmarshal(raw, &i); err == nil {
		return i
	}

	// Try quoted string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.ToLower(strings.TrimSpace(s))

		// Named proc_mode values
		switch s {
		case "auto":
			return 1
		case "manual":
			return 0
		}

		// Named proc_priority values
		switch s {
		case "normal":
			return 0
		case "high":
			return 1
		case "critical":
			return 2
		case "emergency":
			return 3
		}

		// Fallback: numeric string
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}

	return 0
}
