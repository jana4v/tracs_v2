package restapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

type PayloadState struct {
	cfg_no      []int64
	tm_menmonic []string
	state       []string
	priority    []int64
	cost        map[int64]int64
}

var payloadState PayloadState

// var ctx = context.Background()
// var rdb *redis.Client // assuming this is your redis client

func get_info_from_smarttc(r *redis.Client) {
	data, err1 := r.Get(ctx, "data_for_go_func").Result()
	if err1 != nil {
		logger.Println("Failed to read data_for_go_func from Redis database")
		logger.Println(err1)
		return
	}

	cfg_tm_state_priority := strings.Split(data, ";")
	if len(cfg_tm_state_priority) < 4 {
		logger.Println("Invalid data format: expected 4 parts, got", len(cfg_tm_state_priority))
		return
	}

	_cfg_no := strings.Split(cfg_tm_state_priority[0], ",")
	_tm_mnemonic := strings.Split(cfg_tm_state_priority[1], ",")
	_state := strings.Split(cfg_tm_state_priority[2], ",")
	_priority := strings.Split(cfg_tm_state_priority[3], ",")

	// Validate that all arrays have the same length
	minLen := len(_cfg_no)
	if len(_tm_mnemonic) < minLen {
		minLen = len(_tm_mnemonic)
	}
	if len(_state) < minLen {
		minLen = len(_state)
	}
	if len(_priority) < minLen {
		minLen = len(_priority)
	}

	if minLen == 0 {
		logger.Println("No valid data to process")
		return
	}

	// Clear existing data
	payloadState.cfg_no = []int64{}
	payloadState.tm_menmonic = []string{} // Fix typo: tm_menmonic -> tm_mnemonic
	payloadState.state = []string{}
	payloadState.priority = []int64{}
	payloadState.cost = make(map[int64]int64)

	// Process only up to the minimum length
	for i := 0; i < minLen; i++ {
		// Process CFG_NO
		val, err := strconv.ParseInt(_cfg_no[i], 10, 64)
		if err != nil {
			logger.Println("CFG Number problem in converting to number", _cfg_no[i], err)
			continue
		}

		// Process PRIORITY
		priorityVal, err := strconv.ParseInt(_priority[i], 10, 64)
		if err != nil {
			logger.Println("Priority problem in converting to number", _priority[i], err)
			continue
		}

		// Only add if both conversions succeed
		payloadState.cfg_no = append(payloadState.cfg_no, val)
		payloadState.tm_menmonic = append(payloadState.tm_menmonic, _tm_mnemonic[i])
		payloadState.state = append(payloadState.state, _state[i])
		payloadState.priority = append(payloadState.priority, priorityVal)
		payloadState.cost[val] = 0
	}

	logger.Printf("Loaded %d configurations", len(payloadState.cfg_no))
}

func update_payload_state(r *redis.Client) (map[int64][]int64, error) {
	tm_map, err := r.HGetAll(ctx, "TM_MAP").Result()
	if err != nil {
		return nil, err
	}

	// Reset costs to 0
	for cfg := range payloadState.cost {
		payloadState.cost[cfg] = 0
	}

	for i, cfg := range payloadState.cfg_no {
		if i < len(payloadState.tm_menmonic) && i < len(payloadState.state) {
			val := tm_map[payloadState.tm_menmonic[i]]
			if val != payloadState.state[i] {
				payloadState.cost[cfg] = payloadState.cost[cfg] + payloadState.priority[i]
			}
		}
	}

	var costToConfigMap = make(map[int64][]int64)
	var activeConfigs []string // To store configs with cost 0

	for _cfg, _cost := range payloadState.cost {
		if v, ok := costToConfigMap[_cost]; ok {
			v = append(v, _cfg)
			costToConfigMap[_cost] = v
		} else {
			costToConfigMap[_cost] = []int64{_cfg}
		}

		// Collect configs with cost 0
		if _cost == 0 {
			activeConfigs = append(activeConfigs, strconv.FormatInt(_cfg, 10))
		}
	}

	// Set Redis key with comma-separated config numbers that have cost 0
	if len(activeConfigs) > 0 {
		activeConfigsStr := strings.Join(activeConfigs, ",")
		err := r.HSet(ctx, "DTM_MAP", "active_configs", activeConfigsStr).Err()
		if err != nil {
			logger.Println("Failed to set active_configs in Redis:", err)
		}
	} else {
		// If no configs have cost 0, set empty string or handle as needed
		r.HSet(ctx, "DTM_MAP", "active_configs", "")
	}

	return costToConfigMap, nil
}

func handlePayloadStatus(w http.ResponseWriter, r *http.Request) {
	costToConfigMap, err := update_payload_state(rdb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode and send the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(costToConfigMap)
}

// Background task function
func startBackgroundUpdate(r *redis.Client) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Refresh data from Redis every cycle
			//get_info_from_smarttc(r)
			// Update payload state and set active_configs
			_, err := update_payload_state(r)
			if err != nil {
				logger.Println("Error in background update:", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func registerRoutesForGetPayloadState(r *mux.Router) {
	r.HandleFunc("/get_payload_state", handlePayloadStatus).Methods("GET")
}

// Call this function during application startup
func StartPayloadStateUpdater() {
	// Initial data load
	get_info_from_smarttc(rdb)

	// Start background task
	go startBackgroundUpdate(rdb)
}
