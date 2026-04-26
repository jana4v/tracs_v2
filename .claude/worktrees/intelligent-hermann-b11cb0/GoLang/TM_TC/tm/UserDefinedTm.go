package Telemetry

import (
	"context"
	"encoding/json"
	shared "scg/shared"
	"strconv"
	"strings"
	"time"

	"github.com/gammazero/nexus/v3/wamp"
	influxApi "github.com/influxdata/influxdb-client-go/v2/api"
)

func UserDefinedTm() {
	tm_chain := shared.RedisKeys.DERIVED_TM
	tmValueCache := make(map[string]int64)
	upperLimitCache := make(map[string]float64)
	lowerLimitCache := make(map[string]float64)
	limit_check := 0
	analogTmLimitsFromRedis := make(map[string]string)
	digitalTmExpectedValueFromRedis := make(map[string]string)
	var err error
	var tmpkt shared.TmPacket
	var ctx = context.Background()
	var write_api influxApi.WriteAPI
	r := shared.GetRedisConnection()
	enable_influx, _ := r.HGet(ctx, "ENV_VARIABLES", "ENABLE_INFLUX").Result()
	if enable_influx == "YES" {
		write_api = shared.GetInfluxWriteAPI(tm_chain)
	}
	defer r.Close()
	publisher := shared.GetWampConnection()
	defer publisher.Close()
	dataArray := make([][]interface{}, 0)
	counter := 0
	for {
		pipe := r.Pipeline()
		counter++
		if counter == 1 {
			analogTmLimitsFromRedis, _ = r.HGetAll(ctx, shared.RedisKeys.ANALOG_TM_DYNAMIC_LIMITS).Result()
			digitalTmExpectedValueFromRedis, _ = r.HGetAll(ctx, shared.RedisKeys.DIGITAL_TM_DYNAMIC_LIMITS).Result()
		}
		if counter > 5 {
			counter = 0
		}

		data, _ := r.HGetAll(context.Background(), shared.RedisKeys.DERIVED_TM).Result()
		for _, value := range data {
			err = json.Unmarshal([]byte(string(value)), &tmpkt)
			if err != nil {
				continue
			}
			if isFloat(tmpkt.ProcValue) {
				if _, ok := analogTmLimitsFromRedis[tmpkt.Paramid]; ok {
					limits := strings.Split(analogTmLimitsFromRedis[tmpkt.Paramid], ",")
					if ll, ok := stringToFloat(limits[0]); ok {
						tmpkt.LowerLimit = ll
					}
					if ul, ok := stringToFloat(limits[1]); ok {
						tmpkt.UpperLimit = ul
					}
				}
			}

			if tmpkt.ErrDesc != "invalid" {
				writeIfChanged(r, write_api, tm_chain, tmpkt, false, tmValueCache, upperLimitCache, lowerLimitCache)
				if isFloat(tmpkt.ProcValue) {
					pv, _ := strconv.ParseFloat(tmpkt.ProcValue, 64) // Convert to float64
					if pv < tmpkt.LowerLimit {
						limit_check = -1
					} else if pv > tmpkt.UpperLimit {
						limit_check = 1
					} else {
						limit_check = 0
					}
					pipe.HSet(ctx, shared.RedisKeys.TM_MAP, tmpkt.Param, tmpkt.ProcValue).Result()
					dataArray = append(dataArray, []interface{}{tmpkt.Param, tmpkt.ProcValue, tmpkt.LowerLimit, tmpkt.UpperLimit, limit_check})
				} else {
					limit_check = 0
					// If the value cannot be parsed as float, treat it as a discrete value
					if _, ok := digitalTmExpectedValueFromRedis[tmpkt.Paramid]; ok {
						limits := strings.Split(digitalTmExpectedValueFromRedis[tmpkt.Paramid], ",")
						if contains(limits, tmpkt.ProcValue) {
							limit_check = 0
						} else {
							limit_check = 2
						}
					}

					pipe.HSet(ctx, shared.RedisKeys.TM_MAP, tmpkt.Param, tmpkt.ProcValue).Result()
					dataArray = append(dataArray, []interface{}{tmpkt.Param, tmpkt.ProcValue, tmpkt.LowerLimit, tmpkt.UpperLimit, limit_check})
				}
			}
		}
		if len(dataArray) > 0 {
			publisher.Publish(shared.Wamp_user_defined_tm, nil, wamp.List{dataArray}, nil)
			dataArray = [][]interface{}{}
		}
		time.Sleep(2 * time.Second)
		data_kv, _ := r.HGetAll(context.Background(), shared.RedisKeys.DERIVED_TM_KV).Result()
		for key, value := range data_kv {
			pipe.HSet(ctx, shared.RedisKeys.TM_MAP, key, value).Result()
		}
		pipe.Exec(ctx)
		publisher.Publish(shared.Wamp_user_defined_tm_kv, nil, wamp.List{data_kv}, nil)
	}

}
