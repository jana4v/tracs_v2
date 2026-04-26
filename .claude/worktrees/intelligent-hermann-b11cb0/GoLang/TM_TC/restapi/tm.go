package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	shared "scg/shared"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

var ctx = context.Background()
var rdb *redis.Client

func saveToRedis(w http.ResponseWriter, r *http.Request) {
	// Read the body as a byte slice
	var tmpkt shared.TmPacket
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Unmarshal the JSON to get the key-value pairs
	var jsonPayload map[string]interface{}
	if err := json.Unmarshal(body, &jsonPayload); err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// Iterate over the key-value pairs and save each to Redis
	for key, value := range jsonPayload {
		// Convert the value to a JSON string, if it's an object or array,
		// or use directly if it's a string, number, etc.
		var valueToStore string
		if valueBytes, err := json.Marshal(value); err != nil {
			// Handle error if marshaling fails, but continue with other pairs
			fmt.Printf("Failed to marshal value for key '%s': %v\n", key, err)
			continue
		} else {
			valueToStore = string(valueBytes)
			err = json.Unmarshal(valueBytes, &tmpkt)
			if err != nil {
				_, err = rdb.HSet(ctx, shared.RedisKeys.DERIVED_TM_KV, key, valueToStore).Result()
				if err != nil {
					// Handle error if saving to Redis fails, but continue with other pairs
					logger.Printf("Failed to save key '%s' to Redis: %v\n", key, err)
					continue
				}
			} else {
				rdb.HSet(ctx, shared.RedisKeys.DERIVED_TM, key, valueToStore).Result()
				_, err = rdb.HSet(ctx, shared.RedisKeys.DERIVED_TM_KV, key, tmpkt.ProcValue).Result()
				if err != nil {
					// Handle error if saving to Redis fails, but continue with other pairs
					logger.Printf("Failed to save key '%s' to Redis: %v\n", key, err)
					continue
				}
			}
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

type RequestPayload struct {
	Mnemonics []string `json:"mnemonics"`
	Source    string   `json:"source"`
}

type ResponsePayload struct {
	Results map[string]interface{} `json:"results"`
}

// getParamValues fetches the values of mnemonics from Redis based on the source
func getParamValues(mnemonics []string, source string) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	source = strings.ToUpper(source)
	var sources []string
	if source != "" {
		sources = strings.Split(source, ",")
	}

	// Define the Redis maps to search in
	maps := []string{"TM1", "DTM", "PTM1", "PTM2", "TM2", "SMON1", "SMON2", "ADC1", "ADC2"}

	// Iterate over each mnemonic
	for _, mnemonic := range mnemonics {
		mnemonic = strings.ToLower(strings.TrimSpace(mnemonic))
		found := false
		// Check if source is specified
		if len(sources) > 0 {
			// Search only in the specified source map
			for _, source := range sources {
				value, err := rdb.HGet(ctx, source+"_MAP", mnemonic).Result()
				if err == redis.Nil {
					continue
				} else if err != nil {
					results[mnemonic] = ""
					found = true
					break
				} else {
					results[mnemonic] = parseValue(value)
					found = true
					break
				}
			}
		} else {
			// Search across all maps
			for _, m := range maps {
				value, err := rdb.HGet(ctx, m+"_MAP", mnemonic).Result()
				if err == redis.Nil {
					continue
				} else if err != nil {
					results[mnemonic] = ""
				} else {
					results[mnemonic] = parseValue(value)
					found = true
					break
				}
			}
		}

		// If not found in any map, return blank string
		if !found {
			results[mnemonic] = ""
		}
	}

	return results, nil
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	var payload RequestPayload
	var response ResponsePayload
	response.Results = make(map[string]interface{})

	// Decode the request payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results, err := getParamValues(payload.Mnemonics, payload.Source)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response.Results = results
	// Encode and send the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// parseValue tries to parse a value as a number, if not returns it as a string
func parseValue(value string) interface{} {
	// if num, err := strconv.ParseFloat(value, 64); err == nil {
	// 	return num
	// }
	return value
}

func registerRoutesForTm(r *mux.Router) {
	r.HandleFunc("/inject_dtm", saveToRedis).Methods("POST")
	/*
		{
			"TM_MNEMONIC":22
		}
		OR
		{
			"TM_MNEMONIC":{"param_id":"PW1234","param":"mnemonic","source_info":"dtm","raw_count":10,"proc_value":"value"
			"time_stamp":"00:22:12","upper_limit":1,"lower_limit":0,"err_desc":""
			}
		}
	*/
	r.HandleFunc("/get_tm", searchHandler).Methods("POST")
	/*
		Request Payload:
		json
		Copy code
		{
			"mnemonics": ["key1", "key2"],
			"source": "TM1"
		}
		Response Payload:
		json
		Copy code
		{
			"results": {
				"key1": 123.45,
				"key2": "some_string"
			}
		}
	*/
}
