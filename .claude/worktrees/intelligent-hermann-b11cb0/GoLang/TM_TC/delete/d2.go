package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
)

type ExpressionRequest struct {
	Expression string   `json:"expression"`
	Variables  []string `json:"variables"`
	Timeout    int      `json:"timeout,omitempty"` // Timeout in seconds
}

type Expression struct {
	Expression    string
	Variables     []string
	RVariablesMap map[string]string
	OVariablesMap map[string]string
	Timeout       int
}

type Condition struct {
	Condition string
	Status    bool
	Value     string
}

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

func getParamValues(ps Expression) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	var e error
	for k := range ps.RVariablesMap {
		data[k], e = convertInterfaceToString(10)
	}
	return data, e
}

func evaluateExpression(req Expression) ([]Condition, int, error) {
	params := make(map[string]interface{})
	data, err := getParamValues(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	for _, variable := range req.Variables {
		value, _ := convertInterfaceToString(data[variable])
		if intVal, err := strconv.Atoi(value); err == nil {
			params[variable] = intVal
		} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			params[variable] = floatVal
		} else {
			params[variable] = value
		}
	}
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
	response, err := outputTokens(tokens, data)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(result)
	return response, http.StatusOK, nil

}

func extractVariables(expression string) Expression {
	var exp Expression
	exp.OVariablesMap = make(map[string]string)
	exp.RVariablesMap = make(map[string]string)
	exp.Expression = expression
	prefixes := []string{"tm1.", "tm2.", "ptm1.", "ptm2.", "dtm.", "smon.", "adc."}
	var variables []string

	for _, prefix := range prefixes {
		// Match variables in the format: prefix.something
		reSimple := regexp.MustCompile(`\b` + regexp.QuoteMeta(prefix) + `[\w.]+`)
		simpleMatches := reSimple.FindAllString(expression, -1)
		variables = append(variables, simpleMatches...)

		// Match variables in the format: prefix."something with spaces or special characters"
		reComplex := regexp.MustCompile(`\b` + regexp.QuoteMeta(prefix) + `"\s*([^"]+)\s*"`)
		complexMatches := reComplex.FindAllStringSubmatch(expression, -1)
		for _, match := range complexMatches {
			if len(match) > 1 {
				cleaned := prefix + `"` + match[1] + `"`
				variables = append(variables, cleaned)
			}
		}
	}

	// Remove duplicates
	uniqueVariables := make(map[string]bool)
	var result []string
	for _, v := range variables {
		if !uniqueVariables[v] {
			uniqueVariables[v] = true
			result = append(result, v)
		}
	}
	i := 97
	for _, v := range result {
		exp.OVariablesMap[v] = string(rune(i))
		exp.RVariablesMap[string(rune(i))] = v
		exp.Expression = strings.ReplaceAll(exp.Expression, v, string(rune(i)))
		exp.Variables = append(exp.Variables, string(rune(i)))
		i++
	}

	return exp
}

func outputTokens(tokens []govaluate.ExpressionToken, params map[string]interface{}) ([]Condition, error) {
	var conditions []Condition
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if token.Kind == govaluate.CLAUSE {
			continue
		}
		if token.Kind == govaluate.VARIABLE {
			varName := token.Value.(string)
			varValue, exists := params[varName]
			if !exists {
				return nil, fmt.Errorf("variable %s not found in parameters", varName)
			}

			// Process the comparator and the right-hand side value
			if i+2 < len(tokens) {
				comparator := tokens[i+1]
				rightHand := tokens[i+2]
				conditionStr := ""
				expressionStr := ""
				if rightHand.Kind == govaluate.STRING {
					conditionStr = fmt.Sprintf(`%s %s "%v"`, varName, comparator.Value, rightHand.Value)
					expressionStr = fmt.Sprintf(`%v %s "%v"`, varValue, comparator.Value, rightHand.Value)
				} else {
					// Build the condition string
					conditionStr = fmt.Sprintf("%s %s %v", varName, comparator.Value, rightHand.Value)
					expressionStr = fmt.Sprintf("%v %s %v", varValue, comparator.Value, rightHand.Value)
				}

				// Evaluate the condition

				expression, err := govaluate.NewEvaluableExpression(expressionStr)
				if err != nil {
					return nil, err
				}
				result, err := expression.Evaluate(params)
				if err != nil {
					return nil, err
				}

				status := result.(bool)
				valueStr := fmt.Sprintf("%s=%v", varName, varValue)
				condition := Condition{
					Condition: conditionStr,
					Status:    status,
					Value:     valueStr,
				}
				conditions = append(conditions, condition)

				// Skip the next two tokens (comparator and right-hand value)
				i += 2
			}
		}
	}

	return conditions, nil
}
func main() {
	var req ExpressionRequest
	req.Expression = `((tm1."temperature sensor(o/p) value" > 2 && ptm1."pressure_sensor" == 'on') || tm1."x" ==5) && ptm1."y" == 'on'`
	exp := extractVariables(req.Expression)
	fmt.Println(exp.Expression)
	data, x, e := evaluateExpression(exp)
	fmt.Println(data)
	fmt.Println(x)
	if e != nil {
		fmt.Println(e)
	}
}
