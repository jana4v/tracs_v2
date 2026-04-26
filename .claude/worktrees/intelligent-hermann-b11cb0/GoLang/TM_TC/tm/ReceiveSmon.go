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

func SmonSubscriber(strema_name string, ip_port_number string) {
	var conn *ws.Conn
	var err error
	smonCache := make(map[string]string)
	smonDataMap := make(map[string]string)
	var ctx = context.Background()
	re, _ := regexp.Compile(`break`)
	isSmonBreak := false
	force_write := true
	var write_api influxApi.WriteAPI
	r := shared.GetRedisConnection()
	defer r.Close()
	publisher := shared.GetWampConnection()
	defer publisher.Close()
	clear_smon_redis_db(ctx, r, strema_name)
	enable_influx, _ := r.HGet(ctx, "ENV_VARIABLES", "ENABLE_INFLUX").Result()
	if enable_influx == "YES" {
		write_api = shared.GetInfluxWriteAPI("TM1")
	}
	defer r.Close()
	fmt.Println("Hello FROM " + strema_name + " IP_PORT" + ip_port_number)
	customURL := url.URL{Scheme: "ws", Host: ip_port_number, Path: "/ws"}
	for {
		conn, _, err = ws.DefaultDialer.Dial(customURL.String(), nil)
		if err != nil {
			if strema_name == "SMON1" {
				r.Set(ctx, shared.RedisKeys.SMON1_HEART_BEAT, shared.HeartBeatStatus.CONNECTION_FAILED, 0)
				r.Del(ctx, shared.RedisKeys.SMON1_MAP)
			} else if strema_name == "SMON2" {
				r.Set(ctx, shared.RedisKeys.SMON2_HEART_BEAT, shared.HeartBeatStatus.CONNECTION_FAILED, 0)
				r.Del(ctx, shared.RedisKeys.SMON2_MAP)
			}

			logger.Println(err)
			time.Sleep(10 * time.Second)

			continue
		}
		defer conn.Close()
		for {

			if strema_name == "SMON1" {
				r.Set(ctx, shared.RedisKeys.SMON1_HEART_BEAT, shared.HeartBeatStatus.CONNECTED, 0)
			} else if strema_name == "SMON2" {
				r.Set(ctx, shared.RedisKeys.SMON2_HEART_BEAT, shared.HeartBeatStatus.CONNECTED, 0)
			}
			_, msg, err := conn.ReadMessage()

			if err != nil {
				conn.Close()
				break
			}

			if strema_name == "SMON1" {
				r.Set(ctx, shared.RedisKeys.SMON1_HEART_BEAT, shared.HeartBeatStatus.OK, 7*time.Second)
			} else if strema_name == "SMON2" {
				r.Set(ctx, shared.RedisKeys.SMON2_HEART_BEAT, shared.HeartBeatStatus.OK, 7*time.Second)
			}
			var smonPkt shared.ScosPkt

			err = json.Unmarshal([]byte(string(msg)), &smonPkt)
			//fmt.Println(string(msg))
			if err != nil {
				fmt.Println(err)
			}

			isSmonBreak = re.MatchString(smonPkt.Error)
			//fmt.Println(string(msg))
			if isSmonBreak || len(smonPkt.ParamList) == 0 {
				//fmt.Println(string(msg))
				if strema_name == "SMON1" {
					r.Set(ctx, shared.RedisKeys.SMON1_HEART_BEAT, shared.HeartBeatStatus.DATA_BREAK, 0)
					r.Del(ctx, shared.RedisKeys.SMON1_MAP)
				} else if strema_name == "SMON2" {
					r.Set(ctx, shared.RedisKeys.SMON2_HEART_BEAT, shared.HeartBeatStatus.DATA_BREAK, 0)
					r.Del(ctx, shared.RedisKeys.SMON2_MAP)
				}

				if force_write {
					writeSmonIfChanged(r, write_api, strema_name, "", "", true, smonCache)
					force_write = false
					clear_smon_redis_db(ctx, r, strema_name)
				}
				continue
			}

			pipe := r.Pipeline()
			is_smon1 := strema_name == "SMON1"
			smonDataMap = make(map[string]string)
			for _, data := range smonPkt.ParamList {
				_param := strings.ToLower(strings.TrimSpace(data.Mnemonic))
				_val := strings.TrimSpace(data.Value)
				smonDataMap[_param] = _val
				writeSmonIfChanged(r, write_api, strema_name, _param, _val, false, smonCache)
				if is_smon1 {
					pipe.HSet(ctx, shared.RedisKeys.SMON1_MAP, _param, _val)
					pipe.HSet(ctx, shared.RedisKeys.TM_MAP, _param, _val)
				} else {
					pipe.HSet(ctx, shared.RedisKeys.SMON2_MAP, _param, _val)
				}

			}
			_, rerr := pipe.Exec(ctx)
			if rerr != nil {
				panic(rerr)
			}
			if len(smonDataMap) > 0 {
				if strema_name == "SMON1" {
					err = publisher.Publish(shared.Wamp_smon1, nil, wamp.List{smonDataMap}, nil)
					if err != nil {
						logger.Println("Failed to publish WAMP smon1. Error:", err)
					}
				} else if strema_name == "SMON2" {
					err = publisher.Publish(shared.Wamp_smon2, nil, wamp.List{smonDataMap}, nil)
					if err != nil {
						logger.Println("Failed to publish WAMP smon2. Error:", err)
					}
				}
			}
		}
	}

}

func writeSmonIfChanged(r *redis.Client, write_api influxApi.WriteAPI, strema_name string, param string, value string, force_write bool, smonCache map[string]string) {

	if force_write {
		var stored_data map[string]string
		var err error
		if strema_name == "SMON1" {
			stored_data, err = r.HGetAll(context.Background(), shared.RedisKeys.SMON1_MAP).Result()
		} else if strema_name == "SMON2" {
			stored_data, err = r.HGetAll(context.Background(), shared.RedisKeys.SMON2_MAP).Result()
		}
		if err != nil {
			logger.Println(err)
		} else {
			for key, value := range stored_data {
				WriteSmonDataToInfuxDB(write_api, strema_name, key, value)
			}
		}
		return
	} else {

		// Get the last value from the cache
		var is_data_changed bool = false

		if last_value, ok := smonCache[param]; !ok || last_value != value {
			smonCache[param] = value
			is_data_changed = true
		}

		if is_data_changed {
			WriteSmonDataToInfuxDB(write_api, strema_name, param, value)
		}
	}

}

func WriteSmonDataToInfuxDB(write_api influxApi.WriteAPI, strema_name string, param string, value string) {
	// Parse the value as float
	if write_api == nil {
		return
	}
	f, err := strconv.ParseFloat(value, 64)
	var fields map[string]interface{}
	//fmt.Println(data.RawCount)
	if err != nil {
		// If the value cannot be parsed as float, treat it as a discrete value
		fields = map[string]interface{}{
			"discrete_value": value,
		}
	} else {
		//fmt.Println("value is float")
		//fmt.Println(f)
		// If the value can be parsed as float, treat it as an analog value
		fields = map[string]interface{}{
			"analog_value": f,
		}
	}

	tags := map[string]string{
		"source":   "smon",
		"chain":    strema_name,
		"mnemonic": param,
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

func clear_smon_redis_db(ctx context.Context, r *redis.Client, chain string) {
	if chain == "SMON1" {
		r.Del(ctx, shared.RedisKeys.SMON1_MAP)
	} else if chain == "SMON2" {
		r.Del(ctx, shared.RedisKeys.TM2_PKT)
	}
}
