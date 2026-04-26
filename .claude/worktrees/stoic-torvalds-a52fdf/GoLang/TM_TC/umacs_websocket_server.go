package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// Request structures matching the test procedure interface
type LoadProcedureRequest struct {
	Action       string `json:"action"`
	ProcName     string `json:"proc_name"`
	ProcSrc      string `json:"proc_src,omitempty"`
	ProcMode     string `json:"proc_mode,omitempty"`
	ProcPriority string `json:"proc_priority,omitempty"`
}

type CreateProcedureRequest struct {
	ProcName  string `json:"proc_name"`
	Procedure string `json:"procedure"`
}

type ValidateProcedureRequest struct {
	ProcName   string `json:"proc_name"`
	ProcSource string `json:"proc_source"`
	SubSystem  string `json:"subsystem"`
}

type GetStatusRequest struct {
	Action   string `json:"action"`
	ProcName string `json:"proc_name"`
}

// Response structure
type Response struct {
	Ack       bool   `json:"ack"`
	ErrorMsg  string `json:"error_msg,omitempty"`
	ExeStatus string `json:"exe_status,omitempty"`
}

// ProcedureStore tracks procedure statuses
type ProcedureStore struct {
	mu         sync.RWMutex
	procedures map[string]*ProcedureInfo
}

type ProcedureInfo struct {
	Name      string
	Content   string
	Status    string // "queued", "in-progress", "success", "failure"
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
}

var store = &ProcedureStore{
	procedures: make(map[string]*ProcedureInfo),
}

func main1() {
	r := mux.NewRouter()

	// Register endpoints
	r.HandleFunc("/createProcedure", handleCreateProcedure).Methods("POST")
	r.HandleFunc("/validateProcedure", handleValidateProcedure).Methods("POST")
	r.HandleFunc("/loadProcedure", handleLoadProcedure).Methods("POST")
	r.HandleFunc("/getExeStatus", handleGetExeStatus).Methods("POST")

	// CORS middleware
	r.Use(corsMiddleware)

	port := ":8787"
	log.Printf("UMACS WebSocket Mock Server starting on port %s", port)
	log.Printf("Endpoints:")
	log.Printf("  POST /createProcedure")
	log.Printf("  POST /validateProcedure")
	log.Printf("  POST /loadProcedure")
	log.Printf("  POST /getExeStatus")

	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal("Server failed:", err)
	}
}

// CORS middleware
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

// handleCreateProcedure creates a new procedure
func handleCreateProcedure(w http.ResponseWriter, r *http.Request) {
	var req CreateProcedureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, Response{Ack: false, ErrorMsg: "Invalid request payload"})
		return
	}

	log.Printf("Creating procedure: %s", req.ProcName)

	// Simulate validation - randomly fail 10% of the time
	if rand.Float32() < 0.1 {
		log.Printf("Failed to create procedure: %s (simulated error)", req.ProcName)
		sendResponse(w, Response{Ack: false, ErrorMsg: "Failed to create procedure file"})
		return
	}

	store.mu.Lock()
	store.procedures[req.ProcName] = &ProcedureInfo{
		Name:      req.ProcName,
		Content:   req.Procedure,
		Status:    "not-available", // Use valid status instead of "created"
		CreatedAt: time.Now(),
	}
	store.mu.Unlock()

	log.Printf("Procedure created successfully: %s", req.ProcName)
	sendResponse(w, Response{Ack: true})
}

// handleValidateProcedure validates a procedure
func handleValidateProcedure(w http.ResponseWriter, r *http.Request) {
	var req ValidateProcedureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, Response{Ack: false, ErrorMsg: "Invalid request payload"})
		return
	}

	log.Printf("Validating procedure: %s", req.ProcName)

	store.mu.RLock()
	_, exists := store.procedures[req.ProcName]
	store.mu.RUnlock()

	if !exists {
		log.Printf("Procedure not found: %s", req.ProcName)
		sendResponse(w, Response{Ack: false, ErrorMsg: "Procedure not found"})
		return
	}

	// Simulate validation - randomly fail 5% of the time
	if rand.Float32() < 0.05 {
		log.Printf("Validation failed for procedure: %s (simulated error)", req.ProcName)
		sendResponse(w, Response{Ack: false, ErrorMsg: "Procedure validation failed"})
		return
	}

	// Don't change status - keep it as "not-available" until loadProcedure is called
	log.Printf("Procedure validated successfully: %s", req.ProcName)
	sendResponse(w, Response{Ack: true})
}

// handleLoadProcedure loads and executes a procedure
func handleLoadProcedure(w http.ResponseWriter, r *http.Request) {
	var req LoadProcedureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, Response{Ack: false, ErrorMsg: "Invalid request payload"})
		return
	}

	log.Printf("Loading procedure: %s (Mode: %s, Priority: %s)", req.ProcName, req.ProcMode, req.ProcPriority)

	store.mu.RLock()
	proc, exists := store.procedures[req.ProcName]
	store.mu.RUnlock()

	if !exists {
		log.Printf("Procedure not found: %s", req.ProcName)
		sendResponse(w, Response{Ack: false, ErrorMsg: "Procedure not found"})
		return
	}

	// Simulate execution - randomly fail 15% of the time
	if rand.Float32() < 0.15 {
		log.Printf("Failed to load procedure: %s (simulated error)", req.ProcName)
		sendResponse(w, Response{Ack: false, ErrorMsg: "Failed to load procedure"})
		return
	}

	store.mu.Lock()
	proc.Status = "queued"
	proc.StartedAt = time.Now()
	store.mu.Unlock()

	// Simulate asynchronous execution
	go simulateExecution(req.ProcName)

	log.Printf("Procedure queued successfully: %s", req.ProcName)
	sendResponse(w, Response{Ack: true, ExeStatus: "queued"})
}

// handleGetExeStatus returns the execution status of a procedure
func handleGetExeStatus(w http.ResponseWriter, r *http.Request) {
	var req GetStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, Response{Ack: false, ErrorMsg: "Invalid request payload"})
		return
	}

	log.Printf("Getting status for procedure: %s", req.ProcName)

	store.mu.RLock()
	proc, exists := store.procedures[req.ProcName]
	store.mu.RUnlock()

	if !exists {
		// Return "not-available" as ExeStatus when procedure doesn't exist
		// This matches the expected behavior in umacs_tc_file_sts.go
		sendResponse(w, Response{Ack: true, ExeStatus: "not-available"})
		return
	}

	sendResponse(w, Response{Ack: true, ExeStatus: proc.Status})
}

// simulateExecution simulates the execution of a procedure
func simulateExecution(procName string) {
	// Random execution time between 2-8 seconds
	executionTime := time.Duration(2+rand.Intn(6)) * time.Second

	store.mu.Lock()
	proc := store.procedures[procName]
	proc.Status = "in-progress"
	store.mu.Unlock()

	log.Printf("Procedure %s: status changed to in-progress (will take %v)", procName, executionTime)

	time.Sleep(executionTime)

	store.mu.Lock()
	// Randomly succeed (70%) or fail (30%)
	if rand.Float32() < 0.7 {
		proc.Status = "success"
		log.Printf("Procedure %s: completed successfully", procName)
	} else {
		proc.Status = "failure"
		log.Printf("Procedure %s: execution failed", procName)
	}
	proc.EndedAt = time.Now()
	store.mu.Unlock()
}

// sendResponse sends a JSON response
func sendResponse(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Initialize random seed
func init() {
	rand.Seed(time.Now().UnixNano())
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("UMACS TC Mock Server")
	fmt.Println(strings.Repeat("=", 60))
}
