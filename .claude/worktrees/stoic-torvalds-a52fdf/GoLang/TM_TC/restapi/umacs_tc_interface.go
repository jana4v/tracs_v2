package restapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	shared "scg/shared"
	"strings"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var execution_status_failure = []string{"failure", "aborted", "suspended"}
var execution_status_waiting = []string{"queued", "in-progress"}
var execution_status_success = []string{"success"}
var not_available string = "not-available"
var dont_check_status = []string{"failure", "aborted", "suspended", "success"}
var tc_url string = UmacsEnvVariables.TC_API_URL

func contains(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

type Request struct {
	Action       string `json:"action"`
	ProcName     string `json:"proc_name"`
	ProcSrc      string `json:"proc_src,omitempty"`
	ProcMode     string `json:"proc_mode,omitempty"`
	ProcPriority string `json:"proc_priority,omitempty"`
}

type Response struct {
	Ack       bool   `json:"ack"`
	ErrorMsg  string `json:"error_msg,omitempty"`
	ExeStatus string `json:"exe_status,omitempty"`
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

// PQueueTestProcedure represents the structure of test procedure data in the priority queue
type PQueueTestProcedure struct {
	ProcName              string `json:"proc_name"`
	Procedure             string `json:"procedure"`
	ProcSource            string `json:"proc_source,omitempty"`
	SubSystem             string `json:"subsystem,omitempty"`
	ProcPriority          string `json:"proc_priority,omitempty"`
	ProcMode              string `json:"proc_mode,omitempty"`
	OnFail                string `json:"on_fail,omitempty"`
	WaitUntilExecution    string `json:"wait_until_execution,omitempty"`
	RequestedTime         string `json:"requested_time,omitempty"`
	Status                string `json:"status,omitempty"`
	Error                 string `json:"error,omitempty"`
	CompletedTime         string `json:"completed_time,omitempty"`
	TimeTakenForExecution string `json:"time_taken_for_execution,omitempty"`
	RetryCount            int    `json:"retry_count,omitempty"`
	TestPhase             string `json:"test_phase,omitempty"`
}

func (req *Request) trigger_file() (Response, error) {
	req.Action = "loadprocedure"
	tc_url = UmacsEnvVariables.TC_API_URL + "loadProcedure"
	res, err := http_umacs_tc_post_request(req)
	if err != nil {
		return res, err
	} else if !res.Ack {
		return res, fmt.Errorf("negative acknowledgement for loadProcedure \n%s", res.ErrorMsg)
	}
	rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, req.ProcName, "File Triggered in UMACS").Result()
	return res, err
}

func (req *Request) trigger_file_wait_for_execution_complete() (Response, error) {
	req.Action = "loadprocedure"
	tc_url = UmacsEnvVariables.TC_API_URL + "loadProcedure"

	// Add file to Redis BEFORE loading so the poller can track it
	rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, req.ProcName, "queued").Result()

	res, err := http_umacs_tc_post_request(req)
	if err != nil {
		return res, err
	} else if !res.Ack {
		return res, fmt.Errorf("negative acknowledgement for loadProcedure \n%s", res.ErrorMsg)
	}

	for {
		status, err := rdb.HGet(ctx, shared.RedisKeys.TC_FILES_STATUS, req.ProcName).Result()
		if err == redis.Nil || err != nil {
			//log.Printf("File Status Not Available For File: %v", req.ProcName)
			return res, fmt.Errorf("file status not available for file: %v", req.ProcName)
		}
		if contains(status, execution_status_waiting) {
			time.Sleep(time.Second)
		} else if contains(status, execution_status_failure) {
			log.Printf("Execution failed: %v", status)
			return res, fmt.Errorf("execution failed: %v", status)
		} else if contains(status, execution_status_success) {
			return res, nil
		}
	}
}

func (req *Request) Get_file_status() (Response, error) {
	req.Action = "getexestatus"
	tc_url = UmacsEnvVariables.TC_API_URL + "getExeStatus"
	res, err := http_umacs_tc_post_request(req)
	if err != nil {
		return res, err
	} else if !res.Ack {
		return res, fmt.Errorf("negative acknowledgement for getting file status \n%s", res.ErrorMsg)
	} else {
		return res, nil
	}
}

// Define an interface
type PostRequest interface{}

func http_umacs_tc_post_request(req PostRequest) (Response, error) {
	fmt.Println("----------------------------------------------------------------Request")
	fmt.Println(req)
	fmt.Println("----------------------------------------------------------------Request End")
	jsonData, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}
	resp, err := http.Post(tc_url, "application/json", bytes.NewBuffer(jsonData))
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
	fmt.Println("----------------------------------------------------------------Response")
	fmt.Println(externalResponse)
	fmt.Println("----------------------------------------------------------------Response End")
	return externalResponse, nil
}

func (req *CreateProcedure) Create_file_in_umacs() (Response, error) {
	tc_url = UmacsEnvVariables.TC_API_URL + "createProcedure"
	res, err := http_umacs_tc_post_request(req)
	fmt.Println(res.Ack, res.ErrorMsg)
	if err != nil {
		return res, err
	} else if !res.Ack {
		return res, fmt.Errorf("negative acknowledgement for create file in UMACS \n%s", res.ErrorMsg)
	} else {
		return res, nil
	}
}

func (req *ValidateProcedure) Validate_Procedure() (Response, error) {
	tc_url = UmacsEnvVariables.TC_API_URL + "validateProcedure"
	res, err := http_umacs_tc_post_request(req)
	if err != nil {
		return res, err
	} else if !res.Ack {
		return res, fmt.Errorf("negative acknowledgement for validate procedure \n%s", res.ErrorMsg)
	} else {
		return res, nil
	}
}

type TcRestApiRequest struct {
	ProcName  string `json:"proc_name"`
	Procedure string `json:"procedure,omitempty"`
}

func trigger_file_execution(w http.ResponseWriter, r *http.Request) {
	var umacs_env_data = shared.ReadUmacsEnvData(rdb)
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	var areq Request
	areq.ProcName = req.ProcName
	areq.ProcMode = umacs_env_data.TC_API_REQ_EXECUTION_MODE
	areq.ProcPriority = umacs_env_data.TC_API_REQ_PRIORITY
	areq.ProcSrc = umacs_env_data.TC_API_REQ_SOURCE
	res, err := areq.trigger_file()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func trigger_file_execution_and_wait_for_completion(w http.ResponseWriter, r *http.Request) {
	var umacs_env_data = shared.ReadUmacsEnvData(rdb)
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	var areq Request
	areq.ProcName = req.ProcName
	areq.ProcMode = umacs_env_data.TC_API_REQ_EXECUTION_MODE
	areq.ProcPriority = umacs_env_data.TC_API_REQ_PRIORITY
	areq.ProcSrc = umacs_env_data.TC_API_REQ_SOURCE
	res, err := areq.trigger_file_wait_for_execution_complete()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func transfer_file_and_trigger_execution(w http.ResponseWriter, r *http.Request) {
	var umacs_env_data = shared.ReadUmacsEnvData(rdb)
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	var create_req CreateProcedure
	create_req.ProcName = req.ProcName
	create_req.Procedure = req.Procedure
	_, err := create_req.Create_file_in_umacs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	var validate_req ValidateProcedure
	validate_req.ProcName = req.ProcName
	validate_req.ProcSource = umacs_env_data.TC_API_REQ_SOURCE
	validate_req.SubSystem = "PAYLOAD"
	_, err = validate_req.Validate_Procedure()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	var load_request Request
	load_request.ProcName = req.ProcName
	load_request.ProcMode = umacs_env_data.TC_API_REQ_EXECUTION_MODE
	load_request.ProcPriority = umacs_env_data.TC_API_REQ_PRIORITY
	load_request.ProcSrc = umacs_env_data.TC_API_REQ_SOURCE
	res, err := load_request.trigger_file()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func transfer_file_trigger_execution_and_wait_for_completion(w http.ResponseWriter, r *http.Request) {
	var umacs_env_data = shared.ReadUmacsEnvData(rdb)
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	var create_req CreateProcedure
	create_req.ProcName = req.ProcName
	create_req.Procedure = req.Procedure
	_, err := create_req.Create_file_in_umacs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	var validate_req ValidateProcedure
	validate_req.ProcName = req.ProcName
	validate_req.ProcSource = umacs_env_data.TC_API_REQ_SOURCE
	validate_req.SubSystem = "PAYLOAD"
	_, err = validate_req.Validate_Procedure()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	var load_request Request
	load_request.ProcName = req.ProcName
	load_request.ProcMode = umacs_env_data.TC_API_REQ_EXECUTION_MODE
	load_request.ProcPriority = umacs_env_data.TC_API_REQ_PRIORITY
	load_request.ProcSrc = umacs_env_data.TC_API_REQ_SOURCE
	res, err := load_request.trigger_file_wait_for_execution_complete()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func handle_and_execute_split_file(procName, content string) error {
	parts, err := splitAndRenumberFile(content)
	if err != nil {
		return err
	}

	for idx, part := range parts {
		partName := fmt.Sprintf("%s_part%d", procName, idx+1)
		splitContent := strings.Join(part.Lines, "\n") // use .Lines, not .Commands
		// 1. Create file in UMACS, validate, etc.
		createReq := CreateProcedure{ProcName: partName, Procedure: splitContent}
		if _, err := createReq.Create_file_in_umacs(); err != nil {
			log.Printf("Failed to create file in UMACS for part %s: %v", partName, err)
			rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, procName, "failed").Result()
			// If creation fails, we can return the error immediately
			return err
		}
		// validate, etc. as before...
		validateReq := ValidateProcedure{ProcName: partName, ProcSource: UmacsEnvVariables.TC_API_REQ_SOURCE, SubSystem: "PAYLOAD"}
		if _, err := validateReq.Validate_Procedure(); err != nil {
			log.Printf("Failed to validate procedure for part %s: %v", partName, err)
			rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, procName, "failed").Result()
			// If validation fails, we can return the error immediately
			return err
		}

		req := Request{ProcName: partName /* other fields */}
		if _, err := req.trigger_file_wait_for_execution_complete(); err != nil {
			log.Printf("Failed to trigger file execution for part %s: %v", partName, err)
			rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, procName, "failed").Result()
			// If triggering fails, we can return the error immediately
			return err
		}

		// 2. Wait, if required
		if part.WaitAfter > 0 {
			time.Sleep(part.WaitAfter)
		}
	}
	// After all done, mark original file as complete
	rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, procName, "success").Result()
	return nil
}

// create api end point for function handle_and_execute_split_file
// This function will be called by the API endpoint to handle the split file execution
func handle_split_file_execution(w http.ResponseWriter, r *http.Request) {
	var req TcRestApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := handle_and_execute_split_file(req.ProcName, req.Procedure); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Ack: true, ExeStatus: "success"})
}

// ProcessTestProceduresQueue reads from the priority queue and executes test procedures
func ProcessTestProceduresQueue() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("Starting test procedures queue processor...")

	for range ticker.C {
		// Dequeue the next item from the test_procedures queue
		item, err := manager.Dequeue("test_procedures")
		if err != nil {
			log.Printf("Error dequeuing from test_procedures: %v", err)
			continue
		}

		// If queue is empty, continue to next tick
		if item == nil {
			continue
		}

		// Parse the Value field into PQueueTestProcedure struct
		valueBytes, err := json.Marshal(item.Value)
		if err != nil {
			log.Printf("Error marshaling queue item value: %v", err)
			continue
		}

		var testProc PQueueTestProcedure
		err = json.Unmarshal(valueBytes, &testProc)
		if err != nil {
			log.Printf("Error unmarshaling test procedure: %v", err)
			continue
		}

		// Append test phase and timestamp to ProcName
		timestamp := time.Now().Format("2006_01_02T15_04_05.000000")
		originalProcName := testProc.ProcName
		if testProc.TestPhase != "" {
			testProc.ProcName = fmt.Sprintf("%s_%s_%s.tst", originalProcName, testProc.TestPhase, timestamp)
		} else {
			testProc.ProcName = fmt.Sprintf("%s_%s.tst", originalProcName, timestamp)
		}

		log.Printf("Processing test procedure: %s (Original: %s, TestPhase: %s, Priority: %d, Wait: %s)",
			testProc.ProcName, originalProcName, testProc.TestPhase, item.Priority, testProc.WaitUntilExecution)

		// Set initial status and start time
		testProc.Status = "in-progress"
		startTime := time.Now()

		// Execute the procedure
		go executeTestProcedure(testProc, item.Priority, startTime)
	}
}

// executeTestProcedure executes a single test procedure from the queue
func executeTestProcedure(testProc PQueueTestProcedure, priority int, startTime time.Time) {
	var umacs_env_data = shared.ReadUmacsEnvData(rdb)

	// Create file in UMACS
	createReq := CreateProcedure{
		ProcName:  testProc.ProcName,
		Procedure: testProc.Procedure,
	}
	_, err := createReq.Create_file_in_umacs()
	if err != nil {
		log.Printf("Failed to create file in UMACS for %s: %v", testProc.ProcName, err)
		testProc.Status = "failed"
		testProc.Error = fmt.Sprintf("Failed to create file in UMACS: %v", err)
		executionTime := time.Since(startTime)
		testProc.TimeTakenForExecution = executionTime.String()
		testProc.CompletedTime = time.Now().Format(time.RFC3339)
		saveToProcessedQueue(testProc, priority)
		handleFailure(testProc, priority)
		return
	}

	// Validate procedure
	validateReq := ValidateProcedure{
		ProcName:   testProc.ProcName,
		ProcSource: testProc.ProcSource,
		SubSystem:  testProc.SubSystem,
	}
	// Use default values if not provided
	if validateReq.ProcSource == "" {
		validateReq.ProcSource = umacs_env_data.TC_API_REQ_SOURCE
	}
	if validateReq.SubSystem == "" {
		validateReq.SubSystem = "PAYLOAD"
	}

	_, err = validateReq.Validate_Procedure()
	if err != nil {
		log.Printf("Failed to validate procedure for %s: %v", testProc.ProcName, err)
		testProc.Status = "failed"
		testProc.Error = fmt.Sprintf("Failed to validate procedure: %v", err)
		executionTime := time.Since(startTime)
		testProc.TimeTakenForExecution = executionTime.String()
		testProc.CompletedTime = time.Now().Format(time.RFC3339)
		saveToProcessedQueue(testProc, priority)
		handleFailure(testProc, priority)
		return
	}

	// Prepare load request
	loadReq := Request{
		ProcName:     testProc.ProcName,
		ProcMode:     testProc.ProcMode,
		ProcPriority: testProc.ProcPriority,
		ProcSrc:      testProc.ProcSource,
	}
	// Use default values if not provided
	if loadReq.ProcMode == "" {
		loadReq.ProcMode = umacs_env_data.TC_API_REQ_EXECUTION_MODE
	}
	if loadReq.ProcPriority == "" {
		loadReq.ProcPriority = umacs_env_data.TC_API_REQ_PRIORITY
	}
	if loadReq.ProcSrc == "" {
		loadReq.ProcSrc = umacs_env_data.TC_API_REQ_SOURCE
	}

	// Execute based on wait_until_execution flag
	if testProc.WaitUntilExecution == "true" {
		log.Printf("Triggering file and waiting for completion: %s", testProc.ProcName)
		_, err = loadReq.trigger_file_wait_for_execution_complete()
	} else {
		log.Printf("Triggering file without waiting: %s", testProc.ProcName)
		_, err = loadReq.trigger_file()
	}

	if err != nil {
		log.Printf("Failed to trigger file execution for %s: %v", testProc.ProcName, err)
		testProc.Status = "failed"
		testProc.Error = fmt.Sprintf("Failed to trigger file execution: %v", err)
		executionTime := time.Since(startTime)
		testProc.TimeTakenForExecution = executionTime.String()
		testProc.CompletedTime = time.Now().Format(time.RFC3339)
		saveToProcessedQueue(testProc, priority)
		handleFailure(testProc, priority)
		return
	}

	// Success
	testProc.Status = "success"
	testProc.Error = ""
	executionTime := time.Since(startTime)
	testProc.TimeTakenForExecution = executionTime.String()
	testProc.CompletedTime = time.Now().Format(time.RFC3339)
	saveToProcessedQueue(testProc, priority)
	log.Printf("Successfully executed test procedure: %s", testProc.ProcName)
}

// saveToProcessedQueue saves the completed test procedure to the processed_test_procedures queue
func saveToProcessedQueue(testProc PQueueTestProcedure, priority int) {
	item := Item{
		Priority: priority,
		Value:    testProc,
	}
	err := manager.Enqueue("processed_test_procedures", item)
	if err != nil {
		log.Printf("Failed to save processed test procedure %s to queue: %v", testProc.ProcName, err)
	} else {
		log.Printf("Saved processed test procedure %s to processed_test_procedures queue", testProc.ProcName)
	}
}

// handleFailure handles procedure execution failures based on OnFail strategy
func handleFailure(testProc PQueueTestProcedure, priority int) {
	if testProc.OnFail == "retry" && testProc.WaitUntilExecution == "true" && testProc.RetryCount > 0 {
		log.Printf("Scheduling retry for test procedure: %s (remaining retry count: %d)", testProc.ProcName, testProc.RetryCount)
		// Wait 10 seconds before retry
		go func() {
			time.Sleep(10 * time.Second)
			log.Printf("Retrying test procedure after 10 seconds: %s (retry attempt remaining: %d)", testProc.ProcName, testProc.RetryCount)

			// Clean up old procedure state in mock server by deleting from Redis
			// This forces the mock server to treat it as a new procedure
			rdb.HDel(ctx, shared.RedisKeys.TC_FILES_STATUS, testProc.ProcName).Result()

			// Decrement retry count
			testProc.RetryCount--
			// Reset status and error for retry
			testProc.Status = "in-progress"
			testProc.Error = ""
			// Use new start time for retry
			retryStartTime := time.Now()
			executeTestProcedure(testProc, priority, retryStartTime)
		}()
	} else if testProc.OnFail == "retry" && testProc.RetryCount <= 0 {
		log.Printf("Test procedure %s failed, no more retry attempts left (retry count: %d)", testProc.ProcName, testProc.RetryCount)
		// Update status in Redis
		rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, testProc.ProcName, "failure").Result()
	} else {
		log.Printf("Test procedure %s failed, OnFail strategy: %s", testProc.ProcName, testProc.OnFail)
		// Update status in Redis
		rdb.HSet(ctx, shared.RedisKeys.TC_FILES_STATUS, testProc.ProcName, "failure").Result()
	}
}

// StartTestProcedureQueueProcessor starts the background processor for test procedures queue
func StartTestProcedureQueueProcessor() {
	go ProcessTestProceduresQueue()
	log.Println("Test procedure queue processor started")
}

func registerRoutesForUmacsTcInterface(r *mux.Router) {
	r.HandleFunc("/umacs_tc_trigger_file_execution", trigger_file_execution).Methods("POST")
	r.HandleFunc("/umacs_tc_trigger_file_execution_and_wait_for_completion", trigger_file_execution_and_wait_for_completion).Methods("POST")
	r.HandleFunc("/umacs_tc_transfer_file_and_trigger_execution", transfer_file_and_trigger_execution).Methods("POST")
	r.HandleFunc("/umacs_tc_transfer_file_trigger_execution_and_wait_for_completion", transfer_file_trigger_execution_and_wait_for_completion).Methods("POST")
	r.HandleFunc("/umacs_tc_handle_split_file_execution", handle_split_file_execution).Methods("POST")
}
