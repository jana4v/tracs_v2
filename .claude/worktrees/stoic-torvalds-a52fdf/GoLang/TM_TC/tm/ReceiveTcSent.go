package Telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	shared "scg/shared"
	"strings"
	"time"

	"github.com/gammazero/nexus/v3/wamp"
	ws "github.com/gorilla/websocket"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxApi "github.com/influxdata/influxdb-client-go/v2/api"
)

func TcSent(ipPort string) {
	var conn *ws.Conn
	var err error
	var ctx = context.Background()
	var write_api influxApi.WriteAPI
	r := shared.GetRedisConnection()
	enable_influx, _ := r.HGet(ctx, "ENV_VARIABLES", "ENABLE_INFLUX").Result()
	if enable_influx == "YES" {
		write_api = shared.GetInfluxWriteAPI("TC")
	}
	var successRegex = regexp.MustCompile(`success`)
	var tcPkt shared.TcSentPkt
	defer r.Close()
	publisher := shared.GetWampConnection()
	defer publisher.Close()
	customURL := url.URL{Scheme: "ws", Host: ipPort, Path: "/ws"}
	for {
		conn, _, err = ws.DefaultDialer.Dial(customURL.String(), nil)
		if err != nil {
			logger.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		defer conn.Close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				conn.Close()
				fmt.Println(err)
				break
			}

			err = json.Unmarshal([]byte(string(msg)), &tcPkt)
			if err != nil {
				logger.Fatal("Failed to unmarshal tc sent msg:", err)
			}
			err = publisher.Publish(shared.Wamp_tc_sent, nil, wamp.List{tcPkt}, nil)
			if err != nil {
				logger.Fatal("Error in publishing tc sent:", err)
			}
			//fmt.Println(tcPkt)
			//fmt.Println(string(msg))
			WriteTCSentDataToInfuxDB(write_api, tcPkt)
			tcSentCmd := strings.ToLower(strings.TrimSpace(tcPkt.Cmd))
			//tcSentCode := strings.ToLower(strings.TrimSpace(tcPkt.Code))
			//tcSentFullCode := strings.ToLower(strings.TrimSpace(tcPkt.FullCode))
			// tcSentDataCode := strings.ToLower(strings.TrimSpace(tcPkt.DataPart))
			tcSentStatus := strings.ToLower(strings.TrimSpace(tcPkt.Status))
			status := successRegex.MatchString(tcSentStatus)
			if !status {
				continue
			}

			if tcSentCmd != "dataword" {
				r.HSet(ctx, shared.RedisKeys.TM_MAP, "tc_sent", tcSentCmd).Result()
				publish(ctx, r, "TC_SENT_CMD", tcSentCmd)
				publish(ctx, r, "TC_SENT", string(msg))

				//InjectTcDerivedDataCommands(tcSentCmd, tcSentFullCode, r)

				r.LPush(ctx, "tc_sent_list", tcSentCmd)
				cmdsCount := r.LLen(ctx, "tc_sent_list").Val()
				if cmdsCount > 100 {
					r.RPop(ctx, "tc_sent_list")
				}

			}

			// if _, exists := DATA_COMMANDS[tcSentCmd]; exists {
			// 	r.HSet(ctx, RedisKeys.TC_DERIVED_DATA_COMMANDS, tcSentCmd, tcSentFullCode)
			// }
			publisher.Publish(shared.Wamp_tc_sent, nil, wamp.List{tcPkt}, nil)

		}
	}

}

func WriteTCSentDataToInfuxDB(write_api influxApi.WriteAPI, data shared.TcSentPkt) {
	// Parse the value as float
	if write_api == nil {
		return
	}

	fields := map[string]interface{}{
		"cmd":       data.Cmd,
		"code":      data.Code,
		"full_code": data.FullCode,
		"data_part": data.DataPart,
		"status":    data.Status,
	}

	tags := map[string]string{
		"source":   "tc_sent",
		"chain":    "TC",
		"mnemonic": data.Cmd,
	}
	//layout := "02-Jan-2006 15:04:05.000"
	//t := strings.Split(data.TimeStamp, ":")
	//time_stamp := t[0] + ":" + t[1] + ":" + t[2] + "." + t[3]
	//ts, _ := time.Parse(layout, time_stamp)
	// fmt.Println(ts)
	// fmt.Println(time.Now())

	point := influxdb2.NewPoint("telemetry", tags, fields, time.Now())
	write_api.WritePoint(point)
}
