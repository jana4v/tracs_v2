package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	executionStatusFailure = []string{"failure", "aborted", "suspended"}
	executionStatusWaiting = []string{"queued", "in-progress"}
	executionStatusSuccess = []string{"success"}
	notAvailable           = "not-available"
	dontCheckStatus        = []string{"failure", "aborted", "suspended", "success"}
	tcFilesStatusKey       = "TC_FILES_STATUS"
	envVariablesUmacsKey   = "ENV_VARIABLES_UMACS"
)

type Handler struct {
	rdb      *redis.Client
	umacsEnv *UmacsEnvData
	logger   *slog.Logger
}

type Response struct {
	Ack       bool   `json:"ack"`
	ErrorMsg  string `json:"error_msg,omitempty"`
	ExeStatus string `json:"exe_status,omitempty"`
}

type Request struct {
	Action       string `json:"action"`
	ProcName     string `json:"proc_name"`
	ProcSrc      string `json:"proc_src,omitempty"`
	ProcMode     string `json:"proc_mode,omitempty"`
	ProcPriority string `json:"proc_priority,omitempty"`
}

type TcRestApiRequest struct {
	ProcName  string `json:"proc_name"`
	Procedure string `json:"procedure,omitempty"`
}

type CreateProcedure struct {
	ProcName  string `json:"proc_name"`
	Procedure string `json:"procedure"`
}

type ValidateProcedure struct {
	ProcName   string `json:"proc_name"`
	ProcSource string `json:"proc_source"`
	SubSystem  string `json:"subsystem"`
}

type SplitPart struct {
	Lines     []string
	WaitAfter time.Duration
}

func NewHandler(rdb *redis.Client, umacsEnv *UmacsEnvData, logger *slog.Logger) *Handler {
	return &Handler{
		rdb:      rdb,
		umacsEnv: umacsEnv,
		logger:   logger,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/go/v1/umacs_tc_trigger_file_execution", h.triggerFile)
	mux.HandleFunc("POST /api/go/v1/umacs_tc_trigger_file_execution_and_wait_for_completion", h.triggerFileExecutionAndWaitForCompletion)
	mux.HandleFunc("POST /api/go/v1/umacs_tc_transfer_file_and_trigger_execution", h.transferFileAndTriggerExecution)
	mux.HandleFunc("POST /api/go/v1/umacs_tc_transfer_file_trigger_execution_and_wait_for_completion", h.transferFileTriggerExecutionAndWaitForCompletion)
	mux.HandleFunc("POST /api/go/v1/umacs_tc_handle_split_file_execution", h.handleSplitFileExecution)
	mux.HandleFunc("POST /api/go/v1/umacs_tc_enqueue_command", h.enqueueCommand)
	mux.HandleFunc("POST /api/go/v1/umacs_tc_clear_command_queue", h.clearCommandQueue)

	mux.HandleFunc("OPTIONS /api/go/v1/umacs_tc_trigger_file_execution", corsHandler)
	mux.HandleFunc("OPTIONS /api/go/v1/umacs_tc_trigger_file_execution_and_wait_for_completion", corsHandler)
	mux.HandleFunc("OPTIONS /api/go/v1/umacs_tc_transfer_file_and_trigger_execution", corsHandler)
	mux.HandleFunc("OPTIONS /api/go/v1/umacs_tc_transfer_file_trigger_execution_and_wait_for_completion", corsHandler)
	mux.HandleFunc("OPTIONS /api/go/v1/umacs_tc_handle_split_file_execution", corsHandler)
	mux.HandleFunc("OPTIONS /api/go/v1/umacs_tc_enqueue_command", corsHandler)
	mux.HandleFunc("OPTIONS /api/go/v1/umacs_tc_clear_command_queue", corsHandler)
}

func corsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func cors(hfunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		hfunc(w, r)
	}
}

func contains(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func (h *Handler) getTCURL() string {
	return "http://" + h.umacsEnv.TcIP + ":" + h.umacsEnv.TcPort + "/"
}

func (h *Handler) httpUmacsTCPostRequest(req interface{}) (Response, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}

	resp, err := http.Post(h.getTCURL(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	var externalResponse Response
	err = json.Unmarshal(body, &externalResponse)
	if err != nil {
		return Response{}, err
	}

	return externalResponse, nil
}

func (h *Handler) triggerFile(w http.ResponseWriter, r *http.Request) {
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	areq := Request{
		Action:       "loadprocedure",
		ProcName:     req.ProcName,
		ProcMode:     h.umacsEnv.APIReqExecutionMode,
		ProcPriority: h.umacsEnv.APIReqPriority,
		ProcSrc:      h.umacsEnv.APIReqSource,
	}

	res, err := h.triggerFileInternal(areq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) triggerFileInternal(req Request) (Response, error) {
	areq := Request{
		Action:       "loadprocedure",
		ProcName:     req.ProcName,
		ProcMode:     req.ProcMode,
		ProcPriority: req.ProcPriority,
		ProcSrc:      req.ProcSrc,
	}

	tcURL := h.getTCURL() + "loadProcedure"
	jsonData, err := json.Marshal(areq)
	if err != nil {
		return Response{}, err
	}

	resp, err := http.Post(tcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	var externalResponse Response
	err = json.Unmarshal(body, &externalResponse)
	if err != nil {
		return Response{}, err
	}

	if !externalResponse.Ack {
		return externalResponse, fmt.Errorf("negative acknowledgement for loadProcedure: %s", externalResponse.ErrorMsg)
	}

	h.rdb.HSet(context.Background(), tcFilesStatusKey, req.ProcName, "File Triggered in UMACS")
	return externalResponse, nil
}

func (h *Handler) triggerFileExecutionAndWaitForCompletion(w http.ResponseWriter, r *http.Request) {
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	areq := Request{
		ProcName:     req.ProcName,
		ProcMode:     h.umacsEnv.APIReqExecutionMode,
		ProcPriority: h.umacsEnv.APIReqPriority,
		ProcSrc:      h.umacsEnv.APIReqSource,
	}

	res, err := h.triggerFileWaitForExecutionComplete(areq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) triggerFileWaitForExecutionComplete(req Request) (Response, error) {
	h.rdb.HSet(context.Background(), tcFilesStatusKey, req.ProcName, "queued")

	areq := Request{
		Action:       "loadprocedure",
		ProcName:     req.ProcName,
		ProcMode:     req.ProcMode,
		ProcPriority: req.ProcPriority,
		ProcSrc:      req.ProcSrc,
	}

	tcURL := h.getTCURL() + "loadProcedure"
	jsonData, err := json.Marshal(areq)
	if err != nil {
		return Response{}, err
	}

	resp, err := http.Post(tcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	var externalResponse Response
	err = json.Unmarshal(body, &externalResponse)
	if err != nil {
		return Response{}, err
	}

	if !externalResponse.Ack {
		return externalResponse, fmt.Errorf("negative acknowledgement for loadProcedure: %s", externalResponse.ErrorMsg)
	}

	for {
		status, err := h.rdb.HGet(context.Background(), tcFilesStatusKey, req.ProcName).Result()
		if err == redis.Nil || err != nil {
			return Response{}, fmt.Errorf("file status not available for file: %s", req.ProcName)
		}

		if contains(status, executionStatusWaiting) {
			time.Sleep(time.Second)
		} else if contains(status, executionStatusFailure) {
			return Response{}, fmt.Errorf("execution failed: %s", status)
		} else if contains(status, executionStatusSuccess) {
			return externalResponse, nil
		}
	}
}

func (h *Handler) transferFileAndTriggerExecution(w http.ResponseWriter, r *http.Request) {
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	createReq := CreateProcedure{
		ProcName:  req.ProcName,
		Procedure: req.Procedure,
	}

	res, err := h.createFileInUmacs(createReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	validateReq := ValidateProcedure{
		ProcName:   req.ProcName,
		ProcSource: h.umacsEnv.APIReqSource,
		SubSystem:  "PAYLOAD",
	}

	_, err = h.validateProcedure(validateReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	loadReq := Request{
		ProcName:     req.ProcName,
		ProcMode:     h.umacsEnv.APIReqExecutionMode,
		ProcPriority: h.umacsEnv.APIReqPriority,
		ProcSrc:      h.umacsEnv.APIReqSource,
	}

	res, err = h.triggerFileInternal(loadReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) transferFileTriggerExecutionAndWaitForCompletion(w http.ResponseWriter, r *http.Request) {
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	res, err := h.transferFileTriggerExecutionAndWaitForCompletionInternal(req.ProcName, req.Procedure)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) transferFileTriggerExecutionAndWaitForCompletionInternal(procName, procedure string) (Response, error) {
	createReq := CreateProcedure{
		ProcName:  procName,
		Procedure: procedure,
	}

	_, err := h.createFileInUmacs(createReq)
	if err != nil {
		return Response{}, err
	}

	validateReq := ValidateProcedure{
		ProcName:   procName,
		ProcSource: h.umacsEnv.APIReqSource,
		SubSystem:  "PAYLOAD",
	}

	_, err = h.validateProcedure(validateReq)
	if err != nil {
		return Response{}, err
	}

	loadReq := Request{
		ProcName:     procName,
		ProcMode:     h.umacsEnv.APIReqExecutionMode,
		ProcPriority: h.umacsEnv.APIReqPriority,
		ProcSrc:      h.umacsEnv.APIReqSource,
	}

	return h.triggerFileWaitForExecutionComplete(loadReq)
}

func (h *Handler) createFileInUmacs(req CreateProcedure) (Response, error) {
	tcURL := h.getTCURL() + "createProcedure"
	jsonData, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}

	resp, err := http.Post(tcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	var externalResponse Response
	err = json.Unmarshal(body, &externalResponse)
	if err != nil {
		return Response{}, err
	}

	if !externalResponse.Ack {
		return externalResponse, fmt.Errorf("negative acknowledgement for create file in UMACS: %s", externalResponse.ErrorMsg)
	}

	return externalResponse, nil
}

func (h *Handler) validateProcedure(req ValidateProcedure) (Response, error) {
	tcURL := h.getTCURL() + "validateProcedure"
	jsonData, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}

	resp, err := http.Post(tcURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	var externalResponse Response
	err = json.Unmarshal(body, &externalResponse)
	if err != nil {
		return Response{}, err
	}

	if !externalResponse.Ack {
		return externalResponse, fmt.Errorf("negative acknowledgement for validate procedure: %s", externalResponse.ErrorMsg)
	}

	return externalResponse, nil
}

func (h *Handler) handleSplitFileExecution(w http.ResponseWriter, r *http.Request) {
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.handleAndExecuteSplitFile(req.ProcName, req.Procedure); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Ack: true, ExeStatus: "success"})
}

func (h *Handler) handleAndExecuteSplitFile(procName, content string) error {
	parts, err := splitAndRenumberFile(content)
	if err != nil {
		return err
	}

	for idx, part := range parts {
		partName := fmt.Sprintf("%s_part%d", procName, idx+1)
		splitContent := strings.Join(part.Lines, "\n")

		createReq := CreateProcedure{ProcName: partName, Procedure: splitContent}
		if _, err := h.createFileInUmacs(createReq); err != nil {
			h.rdb.HSet(context.Background(), tcFilesStatusKey, procName, "failed")
			return err
		}

		validateReq := ValidateProcedure{ProcName: partName, ProcSource: h.umacsEnv.APIReqSource, SubSystem: "PAYLOAD"}
		if _, err := h.validateProcedure(validateReq); err != nil {
			h.rdb.HSet(context.Background(), tcFilesStatusKey, procName, "failed")
			return err
		}

		req := Request{ProcName: partName}
		if _, err := h.triggerFileWaitForExecutionComplete(req); err != nil {
			h.rdb.HSet(context.Background(), tcFilesStatusKey, procName, "failed")
			return err
		}

		if part.WaitAfter > 0 {
			time.Sleep(part.WaitAfter)
		}
	}

	h.rdb.HSet(context.Background(), tcFilesStatusKey, procName, "success")
	return nil
}

func (h *Handler) enqueueCommand(w http.ResponseWriter, r *http.Request) {
	var req TCCommandPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	procedure := req.Procedure
	if procedure == "" {
		procedure = req.Command
		req.Procedure = procedure
	}
	if req.RequestID == "" || req.ProcedureID == "" || procedure == "" {
		http.Error(w, "request_id, procedure_id, and procedure are required", http.StatusBadRequest)
		return
	}
	if req.Timestamp == "" {
		req.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	payload, err := json.Marshal(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.rdb.ZAdd(r.Context(), TCCommandQueueKey, redis.Z{
		Score:  float64(req.Priority),
		Member: string(payload),
	}).Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{Ack: true})
}

func (h *Handler) clearCommandQueue(w http.ResponseWriter, r *http.Request) {
	if err := h.rdb.Del(r.Context(), TCCommandQueueKey).Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Ack: true})
}

func parseWaitDuration(line string) (time.Duration, error) {
	re := regexp.MustCompile(`wait\s+(\d{2}):(\d{2}):(\d{2}):(\d{3})`)
	matches := re.FindStringSubmatch(line)
	if len(matches) != 5 {
		return 0, fmt.Errorf("invalid wait format: %s", line)
	}
	h, _ := strconv.Atoi(matches[1])
	m, _ := strconv.Atoi(matches[2])
	s, _ := strconv.Atoi(matches[3])
	ms, _ := strconv.Atoi(matches[4])
	totalMs := (((h*60+m)*60 + s) * 1000) + ms
	return time.Duration(totalMs) * time.Millisecond, nil
}

func splitAndRenumberFile(content string) ([]SplitPart, error) {
	lines := strings.Split(content, "\n")
	var parts [][]string
	var waits []time.Duration

	currentPart := []string{}
	waitPattern := regexp.MustCompile(`^(\d{3}\s+)?wait\s+\d{2}:\d{2}:\d{2}:\d{3}`)
	commandPattern := regexp.MustCompile(`^(\d{3})\s+(.*)$`)
	minWait := time.Minute

	for _, line := range lines {
		if waitPattern.MatchString(line) {
			d, err := parseWaitDuration(line)
			if err != nil {
				return nil, err
			}
			if d >= minWait {
				parts = append(parts, currentPart)
				waits = append(waits, d)
				currentPart = []string{}
				continue
			}
		}
		currentPart = append(currentPart, line)
	}

	parts = append(parts, currentPart)
	waits = append(waits, 0)

	var result []SplitPart
	for i, origLines := range parts {
		var outLines []string
		newCmdNum := 1
		for _, line := range origLines {
			if m := commandPattern.FindStringSubmatch(line); m != nil {
				outLines = append(outLines, fmt.Sprintf("%03d %s", newCmdNum, m[2]))
				newCmdNum++
			} else {
				outLines = append(outLines, line)
			}
		}
		outLines = append(outLines, fmt.Sprintf("%03d end", newCmdNum))
		result = append(result, SplitPart{
			Lines:     outLines,
			WaitAfter: waits[i],
		})
	}

	return result, nil
}
