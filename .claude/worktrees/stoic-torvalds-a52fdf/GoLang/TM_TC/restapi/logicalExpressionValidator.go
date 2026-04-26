package restapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/gorilla/mux"
)

func convertInterfaceToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("unsupported type: %T", value)
	}
}

type ExpressionRequest struct {
	Expression string   `json:"expression"`
	Variables  []string `json:"variables"`
	Timeout    int      `json:"timeout,omitempty"` // Timeout in seconds
}

func evaluateExpression(req ExpressionRequest) (map[string]interface{}, int, error) {
	params := make(map[string]interface{})
	values := make(map[string]string)
	data, err := getParamValues(req.Variables, "")
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	for _, variable := range req.Variables {
		value, _ := convertInterfaceToString(data[variable])
		values[variable] = value
		if intVal, err := strconv.Atoi(value); err == nil {
			params[variable] = intVal
		} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			params[variable] = floatVal
		} else {
			params[variable] = value
		}
	}
	fmt.Println(req.Expression, params)
	expression, err := govaluate.NewEvaluableExpression(req.Expression)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	result, err := expression.Evaluate(params)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	// Extract and evaluate each comparison
	tokens := expression.Tokens()
	fmt.Println(tokens)
	conditions := make(map[string]map[string]interface{})

	for _, token := range tokens {
		if token.Kind == govaluate.VARIABLE {
			fmt.Println(token)
			conditions[token.Value.(string)] = map[string]interface{}{
				"condition": fmt.Sprintf("%v == %v", token.Value, values[token.Value.(string)]),
				"status":    params[token.Value.(string)] == values[token.Value.(string)],
				"value":     values[token.Value.(string)],
			}
		}
	}

	response := map[string]interface{}{
		"result":     result,
		"conditions": conditions,
	}
	return response, http.StatusOK, nil

}

func waitUntilExpressionIsTrue(req ExpressionRequest) (map[string]interface{}, int, error) {
	timeout := time.Duration(32) * time.Second
	if req.Timeout != 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ticker.C:
			res, status, err := evaluateExpression(req)
			if err != nil {
				return res, status, err
			} else {
				if res["result"].(bool) {
					return res, status, nil
				}
			}
		case <-timeoutTimer.C:
			return nil, http.StatusRequestTimeout, fmt.Errorf("timeout occurred")
		}
	}

}

func waitUntilExpressionIsTrueHandler(w http.ResponseWriter, r *http.Request) {
	var req ExpressionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	res, status, err := waitUntilExpressionIsTrue(req)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(res)
}

func evaluateExpressionHandler(w http.ResponseWriter, r *http.Request) {
	var req ExpressionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, status, err := evaluateExpression(req)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func registerRoutesForLogicalExpressionEvaluater(r *mux.Router) {
	r.HandleFunc("/expression_evaluate", evaluateExpressionHandler).Methods("POST")
	r.HandleFunc("/wait_until_expression_is_true", waitUntilExpressionIsTrueHandler).Methods("POST")
}
