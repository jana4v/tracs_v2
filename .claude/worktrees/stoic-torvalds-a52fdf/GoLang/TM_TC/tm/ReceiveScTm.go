package Telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	shared "scg/shared"
	"strconv"
	"strings"
	"time"

	"github.com/gammazero/nexus/v3/wamp"
	"github.com/go-redis/redis/v8"
	ws "github.com/gorilla/websocket"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxApi "github.com/influxdata/influxdb-client-go/v2/api"
)

//var TM1_MAP = make(map[string]string)

func TmSubscriber(tm_chain string, ipPort string) {
	fmt.Println("Hello from " + tm_chain)
	tmValueCache := make(map[string]int64)
	upperLimitCache := make(map[string]float64)
	lowerLimitCache := make(map[string]float64)
	analogTmLimitsFromRedis := make(map[string]string)
	digitalTmExpectedValueFromRedis := make(map[string]string)
	fiveSecondCounter := 0
	limit_check := 0
	var ctx = context.Background()
	var conn *ws.Conn
	var req shared.ReqMessage
	var tmpkt shared.TmPacket
	var write_api influxApi.WriteAPI
	publisher := shared.GetWampConnection()
	defer publisher.Close()
	r := shared.GetRedisConnection()
	defer r.Close()
	clear_tm_redis_db(ctx, r, tm_chain)
	enable_influx, _ := r.HGet(ctx, "ENV_VARIABLES", "ENABLE_INFLUX").Result()
	if enable_influx == "YES" {
		write_api = shared.GetInfluxWriteAPI("TM1")
	}
	defer r.Close()
	customURL := url.URL{Scheme: "ws", Host: ipPort, Path: "/ws"}
	// Compile the regular expression once before the loop
	re, err := regexp.Compile(`break`)
	isTMBreak := false
	force_write := true
	if err != nil {
		logger.Fatal(err)
	}

	for {
		conn, _, err = ws.DefaultDialer.Dial(customURL.String(), nil)

		if err != nil {

			if tm_chain == "TM1" {
				r.Set(ctx, shared.RedisKeys.TM1_HEART_BEAT, shared.HeartBeatStatus.CONNECTION_FAILED, 0)
				r.Del(ctx, shared.RedisKeys.TM1_PKT)
				r.Del(ctx, shared.RedisKeys.TM1_MAP)
			} else if tm_chain == "TM2" {
				r.Set(ctx, shared.RedisKeys.TM2_HEART_BEAT, shared.HeartBeatStatus.CONNECTION_FAILED, 0)
				r.Del(ctx, shared.RedisKeys.TM2_PKT)
				r.Del(ctx, shared.RedisKeys.TM2_MAP)
			}
			logger.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		if tm_chain == "TM1" {
			r.Set(ctx, shared.RedisKeys.TM1_HEART_BEAT, shared.HeartBeatStatus.CONNECTED, 0)
		} else if tm_chain == "TM2" {
			r.Set(ctx, shared.RedisKeys.TM2_HEART_BEAT, shared.HeartBeatStatus.CONNECTED, 0)
		}
		defer conn.Close()
		req.Action = "subscribe"
		req.ParamList = []string{""}
		jsonReq, _ := json.Marshal(req)
		conn.WriteMessage(ws.TextMessage, jsonReq)

		dataArray := make([][]interface{}, 0)

		lastSent := time.Now() // Initialize with the current time

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				conn.Close()
				break
			}
			err = json.Unmarshal([]byte(string(msg)), &tmpkt)
			//fmt.Println(string(msg))
			tmpkt.Param = strings.ToLower(strings.TrimSpace(tmpkt.Param))
			tmpkt.ProcValue = strings.ToLower(strings.TrimSpace(tmpkt.ProcValue))
			tmpkt.ErrDesc = strings.ToLower(strings.TrimSpace(tmpkt.ErrDesc))
			isTMBreak = re.MatchString(tmpkt.ErrDesc)

			if isTMBreak {
				//fmt.Println(string(msg))
				if tm_chain == "TM1" {
					r.Set(ctx, shared.RedisKeys.TM1_HEART_BEAT, shared.HeartBeatStatus.DATA_BREAK, 0)
					r.Del(ctx, shared.RedisKeys.TM1_PKT)
					r.Del(ctx, shared.RedisKeys.TM1_MAP)
				} else if tm_chain == "TM2" {
					r.Set(ctx, shared.RedisKeys.TM2_HEART_BEAT, shared.HeartBeatStatus.DATA_BREAK, 0)
					r.Del(ctx, shared.RedisKeys.TM2_PKT)
					r.Del(ctx, shared.RedisKeys.TM2_MAP)
				}

				if force_write {
					writeIfChanged(r, write_api, tm_chain, tmpkt, true, tmValueCache, upperLimitCache, lowerLimitCache)
					force_write = false
					clear_tm_redis_db(ctx, r, tm_chain)
				}
				continue
			}
			if tmpkt.Param == "" {
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

			if tm_chain == "TM1" {
				r.Set(ctx, shared.RedisKeys.TM1_HEART_BEAT, shared.HeartBeatStatus.OK, shared.TmRefreshRate*2*time.Second)
			} else if tm_chain == "TM2" {
				r.Set(ctx, shared.RedisKeys.TM2_HEART_BEAT, shared.HeartBeatStatus.OK, shared.TmRefreshRate*2*time.Second)
			}

			force_write = true
			pipe := r.Pipeline()
			if tm_chain == "TM1" {
				pipe.HSet(ctx, shared.RedisKeys.TM1_PKT, tmpkt.Param, msg)
				pipe.HSet(ctx, shared.RedisKeys.TM1_MAP, tmpkt.Param, tmpkt.ProcValue)
				pipe.HSet(ctx, shared.RedisKeys.TM_MAP, tmpkt.Param, tmpkt.ProcValue)
				// if tmpkt.Param == "l5_ltwta_ch-1_filament_sts" {
				// 	fmt.Println(tmpkt.Param, tmpkt.ProcValue)
				// 	fmt.Println(time.Now().UTC())
				// }
				//fmt.Println(tmpkt.Param, tmpkt.ProcValue)
				//TM1_MAP[tmpkt.Param] = tmpkt.ProcValue
			} else if tm_chain == "TM2" {
				pipe.HSet(ctx, shared.RedisKeys.TM2_PKT, tmpkt.Param, msg)
				pipe.HSet(ctx, shared.RedisKeys.TM2_MAP, tmpkt.Param, tmpkt.ProcValue)
				pipe.HSet(ctx, shared.RedisKeys.TM_MAP, tmpkt.Param, tmpkt.ProcValue)
			}
			_, rerr := pipe.Exec(ctx)
			if rerr != nil {
				panic(err)
			}

			elapsed := time.Since(lastSent)

			// Check if elapsed time is greater than or equal to 1 second
			if elapsed >= 1*time.Second {

				lastSent = time.Now()
				if len(dataArray) > 0 {
					if tm_chain == "TM1" {
						err = publisher.Publish(shared.Wamp_tm1, nil, wamp.List{dataArray}, nil)
						if err != nil {
							logger.Println("Failed to publish WAMP message to TM1. Error: ", err)
						}
					} else if tm_chain == "TM2" {
						err = publisher.Publish(shared.Wamp_tm2, nil, wamp.List{dataArray}, nil)
						if err != nil {
							logger.Println("Failed to publish WAMP message to TM2. Error: ", err)
						}
					}
				}
				dataArray = [][]interface{}{}

				fiveSecondCounter += 1
				if fiveSecondCounter >= 5 {
					fiveSecondCounter = 0
					analogTmLimitsFromRedis, _ = r.HGetAll(ctx, shared.RedisKeys.ANALOG_TM_DYNAMIC_LIMITS).Result()
					digitalTmExpectedValueFromRedis, _ = r.HGetAll(ctx, shared.RedisKeys.DIGITAL_TM_DYNAMIC_LIMITS).Result()
				}

			} else {
				if isFloat(tmpkt.ProcValue) {
					pv, _ := strconv.ParseFloat(tmpkt.ProcValue, 64) // Convert to float64
					// Check value processed value is within the limits or generate an error message
					if pv < tmpkt.LowerLimit {
						limit_check = -1
					} else if pv > tmpkt.UpperLimit {
						limit_check = 1
					} else {
						limit_check = 0
					}
					dataArray = append(dataArray, []interface{}{tmpkt.Param, pv, tmpkt.LowerLimit, tmpkt.UpperLimit, limit_check})
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
					dataArray = append(dataArray, []interface{}{tmpkt.Param, tmpkt.ProcValue, 'D', -1, limit_check})
				}
			}
			// Add data to buffer
			writeIfChanged(r, write_api, tm_chain, tmpkt, false, tmValueCache, upperLimitCache, lowerLimitCache)
		}
	}
}

func writeIfChanged(r *redis.Client, write_api influxApi.WriteAPI, tm_chain string, data shared.TmPacket, force_write bool, tmValueCache map[string]int64, upperLimitCache map[string]float64, lowerLimitCache map[string]float64) {
	if write_api == nil {
		return
	}

	if force_write {
		var stored_data map[string]string
		var err error
		if tm_chain == "TM1" {
			stored_data, err = r.HGetAll(context.Background(), shared.RedisKeys.TM1_PKT).Result()
		} else if tm_chain == "TM2" {
			stored_data, err = r.HGetAll(context.Background(), shared.RedisKeys.TM2_PKT).Result()
		}
		if err != nil {
			logger.Println(err)
		} else {
			for _, value := range stored_data {
				var tmpkt shared.TmPacket
				err = json.Unmarshal([]byte(string(value)), &tmpkt)
				if err != nil {
					logger.Println(err)
				}
				WriteScTelemetryDataToInfuxDB(write_api, tm_chain, tmpkt)
			}
		}
		return
	}

	// Get the last value from the cache
	var is_data_changed bool = false

	if last_raw_count, ok := tmValueCache[data.Param]; !ok || last_raw_count != data.RawCount {
		tmValueCache[data.Param] = data.RawCount
		is_data_changed = true
	}

	if upperLimit, ok := upperLimitCache[data.Param]; !ok || upperLimit != data.UpperLimit {
		upperLimitCache[data.Param] = data.UpperLimit
		is_data_changed = true
	}
	if lowerLimit, ok := lowerLimitCache[data.Param]; !ok || lowerLimit != data.LowerLimit {
		lowerLimitCache[data.Param] = data.LowerLimit
		is_data_changed = true
	}

	if is_data_changed {
		WriteScTelemetryDataToInfuxDB(write_api, tm_chain, data)
	}

}

func WriteScTelemetryDataToInfuxDB(write_api influxApi.WriteAPI, tm_chain string, data shared.TmPacket) {
	// Parse the value as float
	if write_api == nil {
		return
	}
	f, err := strconv.ParseFloat(data.ProcValue, 64)
	var fields map[string]interface{}
	//fmt.Println(data.RawCount)
	if err != nil {
		// If the value cannot be parsed as float, treat it as a discrete value
		fields = map[string]interface{}{
			"discrete_value": data.ProcValue,
			"raw_count":      data.RawCount,
			"upper_limit":    data.UpperLimit,
			"lower_limit":    data.LowerLimit,
		}
	} else {
		//fmt.Println("value is float")
		//fmt.Println(f)
		// If the value can be parsed as float, treat it as an analog value
		fields = map[string]interface{}{
			"analog_value": f,
			"raw_count":    data.RawCount,
			"upper_limit":  data.UpperLimit,
			"lower_limit":  data.LowerLimit,
		}
	}

	tags := map[string]string{
		"source":   "satellite",
		"chain":    tm_chain,
		"mnemonic": data.Param,
	}
	//layout := "02-Jan-2006 15:04:05.000"
	//	t := strings.Split(data.TimeStamp, ":")
	//time_stamp := t[0] + ":" + t[1] + ":" + t[2] + "." + t[3]
	//ts, _ := time.Parse(layout, time_stamp)
	// fmt.Println(ts)
	// fmt.Println(time.Now())
	point := influxdb2.NewPoint("telemetry", tags, fields, time.Now())
	write_api.WritePoint(point)
}

func clear_tm_redis_db(ctx context.Context, r *redis.Client, chain string) {
	pipe := r.Pipeline()
	if chain == "TM1" {
		pipe.Del(ctx, shared.RedisKeys.TM1_PKT)
		pipe.Del(ctx, shared.RedisKeys.TM1_MAP)

	} else if chain == "TM2" {
		pipe.Del(ctx, shared.RedisKeys.TM2_PKT)
		pipe.Del(ctx, shared.RedisKeys.TM2_MAP)

	}
	pipe.Exec(ctx)
}
