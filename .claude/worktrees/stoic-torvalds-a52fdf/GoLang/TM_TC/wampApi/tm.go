package wampApi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	shared "scg/shared"
	"strings"

	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var rdb *redis.Client

// saveToRedis - WAMP equivalent of injecting DTM
func saveToRedisHandler(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	var tmpkt shared.TmPacket

	if len(inv.Arguments) == 0 {
		return formatResponse(nil, fmt.Errorf("missing payload"))
	}

	payload, ok := inv.Arguments[0].(map[string]interface{})
	if !ok {
		return formatResponse(nil, fmt.Errorf("invalid payload format"))
	}

	for key, value := range payload {
		valueBytes, err := json.Marshal(value)
		if err != nil {
			log.Printf("Failed to marshal value for key '%s': %v", key, err)
			continue
		}

		if err = json.Unmarshal(valueBytes, &tmpkt); err != nil {
			_, err = rdb.HSet(ctx, shared.RedisKeys.DERIVED_TM_KV, key, string(valueBytes)).Result()
			if err != nil {
				log.Printf("Failed to save key '%s' to Redis: %v", key, err)
				continue
			}
		} else {
			rdb.HSet(ctx, shared.RedisKeys.DERIVED_TM, key, string(valueBytes)).Result()
			_, err = rdb.HSet(ctx, shared.RedisKeys.DERIVED_TM_KV, key, tmpkt.ProcValue).Result()
			if err != nil {
				log.Printf("Failed to save key '%s' to Redis: %v", key, err)
				continue
			}
		}
	}

	return formatResponse("OK", nil)
}

// getParamValues fetches values from Redis
func getParamValues(mnemonics []string, source string) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	source = strings.ToUpper(source)
	var sources []string
	if source != "" {
		sources = strings.Split(source, ",")
	}

	maps := []string{"DTM", "PTM1", "TM1", "PTM2", "TM2", "SMON1", "SMON2", "ADC1", "ADC2"}

	for _, mnemonic := range mnemonics {
		mnemonic = strings.ToLower(strings.TrimSpace(mnemonic))
		found := false

		if len(sources) > 0 {
			for _, source := range sources {
				value, err := rdb.HGet(ctx, source+"_MAP", mnemonic).Result()
				if err == redis.Nil {
					continue
				} else if err != nil {
					results[mnemonic] = ""
					found = true
					break
				} else {
					results[mnemonic] = value
					found = true
					break
				}
			}
		} else {
			for _, m := range maps {
				value, err := rdb.HGet(ctx, m+"_MAP", mnemonic).Result()
				if err == redis.Nil {
					continue
				} else if err != nil {
					results[mnemonic] = ""
				} else {
					results[mnemonic] = value
					found = true
					break
				}
			}
		}

		if !found {
			results[mnemonic] = ""
		}
	}

	return results, nil
}

func getTmHandler(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	if len(inv.Arguments) == 0 {
		return formatResponse(nil, fmt.Errorf("missing payload"))
	}

	payload, ok := inv.Arguments[0].(map[string]interface{})
	if !ok {
		return formatResponse(nil, fmt.Errorf("invalid payload format"))
	}
	mnemonicsInterface, mnemonicsOk := payload["mnemonics"].([]interface{})
	if !mnemonicsOk {
		return formatResponse(nil, fmt.Errorf("invalid mnemonics format"))
	}

	mnemonics := make([]string, len(mnemonicsInterface))
	for i, v := range mnemonicsInterface {
		str, ok := v.(string)
		if !ok {
			return formatResponse(nil, fmt.Errorf("invalid value in mnemonics list"))
		}
		mnemonics[i] = str
	}
	//mnemonics, mnemonicsOk := payload["mnemonics"].([]string)
	source, _ := payload["source"].(string)

	results, err := getParamValues(mnemonics, source)
	if err != nil {
		return client.InvokeResult{Args: wamp.List{fmt.Sprintf("Error retrieving data: %v", err)}}
	}

	return formatResponse(results, nil)
}

func getTmMnemonisHandler(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	// if len(inv.Arguments) == 0 {
	// 	return client.InvokeResult{Args: wamp.List{"Missing payload"}}
	// }

	// payload, ok := inv.Arguments[0].(map[string]interface{})
	// if !ok {
	// 	return client.InvokeResult{Args: wamp.List{"Invalid payload format"}}
	// }

	// tm_type, tm_typeOk := payload["mnemonics"].(string)

	// if !tm_typeOk {
	// 	return client.InvokeResult{Args: wamp.List{"Invalid input parameters"}}
	// }

	result, err := rdb.HGetAll(ctx, shared.RedisKeys.TM_MAP).Result()
	var keys []string
	for key := range result {
		keys = append(keys, key)
	}
	if err != nil {

		return formatResponse(nil, fmt.Errorf("error retrieving data: %v", err))
	}
	return formatResponse(keys, nil)
}

func RegisterTMApiProcedures(wampArangoClient *client.Client) {
	rdb = shared.GetRedisConnection()
	var isConnected bool = initDB()
	if isConnected {

		// Register all database operations
		wampArangoClient.Register("scg.tm.inject_dtm", saveToRedisHandler, nil)
		wampArangoClient.Register("scg.tm.get_tm_data", getTmHandler, nil)
		wampArangoClient.Register("scg.tm.get_tm_mnemonics", getTmMnemonisHandler, nil)

	}
}
