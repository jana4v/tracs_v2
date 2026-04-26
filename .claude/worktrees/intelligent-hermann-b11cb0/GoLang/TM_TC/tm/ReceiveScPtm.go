package Telemetry

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	shared "scg/shared"
	"strconv"
	"strings"
	"time"

	"github.com/gammazero/nexus/v3/wamp"
	ws "github.com/gorilla/websocket"
)

func PtmSubscriber(tm_chain string, ipPort string) {
	var paramName string
	var paramProcValue string
	var tempUlimit float64
	var tempLlimit float64

	analogTmLimitsFromRedis := make(map[string]string)
	digitalTmExpectedValueFromRedis := make(map[string]string)
	fiveSecondCounter := 0
	limit_check := 0
	var ctx = context.Background()
	var conn *ws.Conn
	var req shared.ReqMessage
	var tmpkt shared.PTmPacket

	publisher := shared.GetWampConnection()
	defer publisher.Close()
	r := shared.GetRedisConnection()
	defer r.Close()
	clear_tm_redis_db(ctx, r, tm_chain)
	defer r.Close()
	customURL := url.URL{Scheme: "ws", Host: ipPort, Path: "/ws"}
	// Compile the regular expression once before the loop
	re, err := regexp.Compile(`break`)
	isTMBreak := false

	if err != nil {
		logger.Fatal(err)
	}

	for {
		conn, _, err = ws.DefaultDialer.Dial(customURL.String(), nil)
		if err != nil {
			if tm_chain == "PTM1" {
				r.Set(ctx, shared.RedisKeys.PTM1_HEART_BEAT, shared.HeartBeatStatus.CONNECTION_FAILED, 0)
				r.Del(ctx, shared.RedisKeys.PTM1_MAP)
			} else if tm_chain == "PTM2" {
				r.Set(ctx, shared.RedisKeys.PTM2_HEART_BEAT, shared.HeartBeatStatus.CONNECTION_FAILED, 0)
				r.Del(ctx, shared.RedisKeys.PTM2_MAP)
			}
			logger.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		if tm_chain == "PTM1" {
			r.Set(ctx, shared.RedisKeys.PTM1_HEART_BEAT, shared.HeartBeatStatus.CONNECTED, 0)
		} else if tm_chain == "PTM2" {
			r.Set(ctx, shared.RedisKeys.PTM2_HEART_BEAT, shared.HeartBeatStatus.CONNECTED, 0)
		}
		defer conn.Close()
		req.Action = "subscribe"
		req.ParamList = []string{""}
		jsonReq, _ := json.Marshal(req)
		conn.WriteMessage(ws.TextMessage, jsonReq)

		dataArray := make([][]interface{}, 0)
		r.Del(ctx, shared.RedisKeys.PTM1_MAP)
		r.Del(ctx, shared.RedisKeys.PTM2_MAP)

		for {

			_, msg, err := conn.ReadMessage()
			if err != nil {
				conn.Close()
				break
			}

			fiveSecondCounter += 1
			if fiveSecondCounter >= 5 {
				fiveSecondCounter = 0
				analogTmLimitsFromRedis, _ = r.HGetAll(ctx, shared.RedisKeys.ANALOG_TM_DYNAMIC_LIMITS).Result()
				digitalTmExpectedValueFromRedis, _ = r.HGetAll(ctx, shared.RedisKeys.DIGITAL_TM_DYNAMIC_LIMITS).Result()
			}

			err = json.Unmarshal([]byte(string(msg)), &tmpkt)

			isTMBreak = re.MatchString(tmpkt.ErrDesc)
			if isTMBreak {
				if tm_chain == "PTM1" {
					r.Set(ctx, shared.RedisKeys.PTM1_HEART_BEAT, shared.HeartBeatStatus.DATA_BREAK, 0)
					r.Del(ctx, shared.RedisKeys.PTM1_MAP)
				} else if tm_chain == "PTM2" {
					r.Set(ctx, shared.RedisKeys.PTM2_HEART_BEAT, shared.HeartBeatStatus.DATA_BREAK, 0)
					r.Del(ctx, shared.RedisKeys.PTM2_MAP)
				}
				continue
			}
			pipe := r.Pipeline()
			for i, param := range tmpkt.Param {
				paramName = strings.ToLower(strings.TrimSpace(param))
				paramProcValue = strings.ToLower(strings.TrimSpace(tmpkt.ProcValue[i]))
				tempUlimit = tmpkt.UpperLimit[i]
				tempLlimit = tmpkt.LowerLimit[i]
				if isFloat(paramProcValue) {
					if _, ok := analogTmLimitsFromRedis[paramName]; ok {
						limits := strings.Split(analogTmLimitsFromRedis[paramName], ",")
						if ll, ok := stringToFloat(limits[0]); ok {
							tempLlimit = ll
						}
						if ul, ok := stringToFloat(limits[1]); ok {
							tempUlimit = ul
						}
					}
					pv, _ := strconv.ParseFloat(paramProcValue, 64) // Convert to float64
					if pv < tempLlimit {
						limit_check = -1
					} else if pv > tempUlimit {
						limit_check = 1
					} else {
						limit_check = 0
					}
					dataArray = append(dataArray, []interface{}{paramName, paramProcValue, tempLlimit, tempUlimit, limit_check})
				} else {
					limit_check = 0
					if _, ok := digitalTmExpectedValueFromRedis[paramName]; ok {
						limits := strings.Split(digitalTmExpectedValueFromRedis[paramName], ",")
						if contains(limits, paramProcValue) {
							limit_check = 0
						} else {
							limit_check = 2
						}
					}
					dataArray = append(dataArray, []interface{}{paramName, paramProcValue, 'D', -1, limit_check})
				}
				if tm_chain == "PTM1" {
					pipe.HSet(ctx, shared.RedisKeys.PTM1_MAP, paramName, paramProcValue)
				} else if tm_chain == "PTM2" {
					pipe.HSet(ctx, shared.RedisKeys.PTM2_MAP, paramName, paramProcValue)
				}
			}
			_, rerr := pipe.Exec(ctx)
			if rerr != nil {
				panic(err)
			}
			if len(dataArray) > 0 {
				if tm_chain == "PTM1" {
					r.Set(ctx, shared.RedisKeys.PTM1_HEART_BEAT, shared.HeartBeatStatus.OK, shared.TmRefreshRate*2*time.Second)
					err = publisher.Publish(shared.Wamp_ptm1, nil, wamp.List{dataArray}, nil)
					if err != nil {
						logger.Println("Failed to publish WAMP message to PTM1. Error: ", err)
					}
				} else if tm_chain == "PTM2" {
					r.Set(ctx, shared.RedisKeys.PTM2_HEART_BEAT, shared.HeartBeatStatus.OK, shared.TmRefreshRate*2*time.Second)
					err = publisher.Publish(shared.Wamp_ptm2, nil, wamp.List{dataArray}, nil)
					if err != nil {
						logger.Println("Failed to publish WAMP message to PTM2. Error: ", err)
					}
				}
			}
			dataArray = [][]interface{}{}
		}

	}
}
