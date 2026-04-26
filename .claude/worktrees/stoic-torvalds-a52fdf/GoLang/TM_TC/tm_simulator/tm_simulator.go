package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"

	"math/rand"
	"net"
	"net/http"
	log "scg/logs"
	shared "scg/shared"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/alexbrainman/odbc"
	redis "github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	ws "github.com/gorilla/websocket"
)

var logger = log.Logger

type tmParam struct {
	TM_MNEMONIC    string
	PossibleStates string
	TM_TYPE        string
}

type clientInfo struct {
	connection    *ws.Conn
	subscriptions []string
}

type subscriptionInfo struct {
	Action string   `json:"action"`
	Params []string `json:"params"`
}
type injectedTm struct {
	Mnemonic   string  `json:"mnemonic"`
	Value      string  `json:"value"`
	LowerLimit float64 `json:"lower_limit"`
	UpperLimit float64 `json:"upper_limit"`
}
type injectedTC struct {
	Cmd      string `json:"cmd"`
	Code     string `json:"code"`
	FullCode string `json:"full_code"`
	DataPart string `json:"data_part"`
	Status   string `json:"status"`
}

var tm_mnemonic_simulated_data_map = make(map[string]string)
var digital_tm_map map[string][]string
var analog_tm_map map[string][2]float64

var wsTM1 *string
var tmType *bool

var chain1Clients []*clientInfo
var TCClients []*clientInfo

// var chain1Subscriptions []string
var upgrader = websocket.Upgrader{} // use default options
var wg sync.WaitGroup
var ctx = context.Background()

func main() {
	wsTM1 = flag.String("wstm1", "9050", "Websocket TM-1 Publish Port")
	tmType = flag.Bool("random", false, "Simulate random telemetry")
	flag.Parse()
	fmt.Println(*wsTM1, *tmType)
	r := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	defer r.Close()
	_, err := r.Ping(ctx).Result()
	if err != nil {
		logger.Println("Failed to Connect Redis Server")
		logger.Println(err)
	}
	if err != nil {
		logger.Println("Unable to set injected_tm_from_tc key in redis database")
		logger.Println(err)
	}
	r.Del(ctx, "INJECTED_TC_FOR_SIMULATOR_MAP")
	r.Del(ctx, "INJECTED_TM_FOR_SIMULATOR_MAP")
	r.Del(ctx, shared.RedisKeys.DERIVED_TM_KV)
	r.Del(ctx, shared.RedisKeys.TM_MAP)

	r.HSet(ctx, "SOFTWARE_CFG", "ENABLE_INJECT_TM", "1").Result()

	wg.Add(1)
	getTelemetryMnemonics()
	simulateFixedTm()
	go startTM1Publisher()
	go startTCPublisher()

	go func() {
		r := mux.NewRouter()
		r.HandleFunc("/", homePage)
		r.HandleFunc("/ws", wsTM1EndPoint)
		logger.Fatal(http.ListenAndServe(":"+*wsTM1, r))
	}()

	go func() {
		r := mux.NewRouter()
		r.HandleFunc("/", homePage)
		r.HandleFunc("/ws", wsTCEndPoint)
		logger.Fatal(http.ListenAndServe(":9070", r))
	}()

	//doEvery(1*time.Second, startTM1Publisher)
	wg.Wait()

}

func getTelemetryMnemonics() {
	db, err := sql.Open("odbc", "DSN=TM_TC")
	r := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 0})
	defer r.Close()
	if err != nil {
		logger.Fatal("Databse Connection error:", err)
	}
	defer db.Close()
	result, err := db.Query("select TM_MNEMONIC,PossibleStates,TM_TYPE from TMTBL")
	if err != nil {
		fmt.Println("     ERROR:", err)
	}

	defer result.Close()
	var param tmParam
	digital_tm_map = make(map[string][]string)
	analog_tm_map = make(map[string][2]float64)
	var _tm_mnemonic string
	var _tm_values []string
	for result.Next() {
		err := result.Scan(&param.TM_MNEMONIC, &param.PossibleStates, &param.TM_TYPE)
		if err != nil {
			fmt.Println(err)
		}
		_tm_mnemonic = strings.TrimSpace(strings.ToLower(param.TM_MNEMONIC))
		if param.TM_TYPE == "NORMAL" {
			tm_mnemonic_simulated_data_map[_tm_mnemonic] = ""
		}
		if res := strings.Contains(param.PossibleStates, ";"); res {
			_tm_values = strings.Split(param.PossibleStates, ";")
			if len(_tm_values) == 2 {
				_start, e1 := strconv.ParseFloat(_tm_values[0], 64)
				_end, e2 := strconv.ParseFloat(_tm_values[1], 64)
				if (e1 == nil) && (e2 == nil) {
					if param.TM_TYPE == "NORMAL" {
						analog_tm_map[_tm_mnemonic] = [2]float64{_start, _end}
					}
					if param.TM_TYPE == "INJECTED" {
						r.HSet(ctx, shared.RedisKeys.DERIVED_TM_KV, _tm_mnemonic, _start).Result()
					}
					//fmt.Println([2]float64{_start, _end})
				} else {
					fmt.Println("Error in possible states in TMTBL database table with tmmnemonic:", param.TM_MNEMONIC)
					fmt.Println(e1, e2)
				}

			} else {
				fmt.Println("Error in possible states in TMTBL database table with tmmnemonic:", param.TM_MNEMONIC)

			}
		}
		if res := strings.Contains(param.PossibleStates, ","); res {
			_tm_values = strings.Split(param.PossibleStates, ",")
			if len(_tm_values) >= 2 {
				for i, val := range _tm_values {
					_tm_values[i] = strings.TrimSpace(strings.ToLower(val))
				}
				if param.TM_TYPE == "NORMAL" {
					digital_tm_map[_tm_mnemonic] = _tm_values
				}
				if param.TM_TYPE == "INJECTED" {
					r.HSet(ctx, shared.RedisKeys.DERIVED_TM_KV, _tm_mnemonic, _tm_values[0]).Result()
				}

			} else {
				fmt.Println("Error in possible states in TMTBL database table with tmmnemonic:", param.TM_MNEMONIC)
			}
		}
	}
}

func simulateRandomTm() {
	for k, v := range analog_tm_map {
		tm_mnemonic_simulated_data_map[k] = fmt.Sprintf("%f", (rand.Float64()*(v[1]-v[0]))+v[0])
		//fmt.Println(rand.Float64(), (v[1] - v[0]), v[0], k)

	}
	for k, v := range digital_tm_map {
		tm_mnemonic_simulated_data_map[k] = v[rand.Intn(len(v))]
	}
}
func simulateFixedTm() {
	for k, v := range analog_tm_map {
		tm_mnemonic_simulated_data_map[k] = fmt.Sprintf("%f", v[0])
	}
	for k, v := range digital_tm_map {
		// var sv = v[0]
		// if sv == "on" {
		// 	sv = "off"
		// }
		// if sv == "enable" {
		// 	sv = "disable"
		// }
		// if sv == "done" {
		// 	sv = "not-done"
		// }
		// if sv == "ready" {
		// 	sv = "not-ready"
		// }
		tm_mnemonic_simulated_data_map[k] = v[0]
	}

}

func getInjectedTm(r *redis.Client) {
	enable_inject_tm, err := r.HGet(ctx, "SOFTWARE_CFG", "ENABLE_INJECT_TM").Result()
	if err != nil {
		r.HSet(ctx, "SOFTWARE_CFG", "ENABLE_INJECT_TM", "1").Result()
		logger.Println("Error getting ENABLE_INJECT_TM from Redis:", err)
		enable_inject_tm = "1" // Default to "1" if not set
	}
	if enable_inject_tm == "0" {
		return
	}
	injected_data := r.HGetAll(ctx, "INJECTED_TM_FOR_SIMULATOR_MAP").Val()
	for key, value := range injected_data {

		if _, exists := tm_mnemonic_simulated_data_map[key]; exists {
			tm_mnemonic_simulated_data_map[key] = value
		} else {
			r.HSet(ctx, shared.RedisKeys.DERIVED_TM_KV, key, value).Result()

		}
	}
}

func startTM1Publisher() {

	r := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 0})
	defer r.Close()
	jsonPkt := shared.TmPacket{}
	//simulateRandomTm()
	for {
		if *tmType {
			simulateRandomTm()
		} else {
			getInjectedTm(r)

		}

		for k, v := range tm_mnemonic_simulated_data_map {
			jsonPkt.Param = k
			jsonPkt.RawCount = -1
			jsonPkt.ProcValue = v
			jsonPkt.SourceInfo = "sim"
			jsonPkt.TimeStamp = "sim"
			if val, ok := analog_tm_map[k]; ok {
				jsonPkt.UpperLimit = val[1]
				jsonPkt.LowerLimit = val[0]
			} else {
				jsonPkt.UpperLimit = -1
				jsonPkt.LowerLimit = -1
			}
			jsonPkt.ErrDesc = ""
			fmtJSON, _ := json.Marshal(jsonPkt)
			isPrblm := false
			for _, client := range chain1Clients {
				err := client.connection.WriteMessage(ws.TextMessage, fmtJSON)
				//fmt.Println(jsonPkt)
				if err != nil {
					isPrblm = true
				}
			}

			if isPrblm {
				removeChain1Client()
			}
			//time.Sleep(10 * time.Nanosecond)
		}
		time.Sleep(500 * time.Millisecond)

	}

}

func homePage(w http.ResponseWriter, r *http.Request) {
	//ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	//fmt.Fprintf(w, "Nothing Here! Go back %+v", ip)
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")

}

func wsTM1EndPoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
		return
	}
	client := &clientInfo{}
	client.connection = conn
	chain1Clients = append(chain1Clients, client)
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	fmt.Println("New client connected from", ip, "to Chain-1")
	fmt.Println("No. of Chain-1 Clients: ", len(chain1Clients))
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Println(err)
			return
		}
		handleReceivedMessage(client, 1, msg)
	}
}

func removeChain1Client() {
	for index, client := range chain1Clients {
		if err := client.connection.WriteMessage(ws.PingMessage, []byte{}); err != nil {
			fmt.Println(err)
			chain1Clients = append(chain1Clients[:index], chain1Clients[index+1:]...)
			removeChain1Client()
			break
		}
	}
}

func handleReceivedMessage(client *clientInfo, chain int, msg []byte) {

	var jsonPkt subscriptionInfo
	err := json.Unmarshal(msg, &jsonPkt)
	if err != nil {
		fmt.Println("Received Wrong Packet:", err)
	}

	if jsonPkt.Action == "subscribe" {

		for i := 0; i < len(jsonPkt.Params); i++ {
			if jsonPkt.Params[i] == "" {
				client.subscriptions = append(client.subscriptions, "*")

			} else {
				param := strings.ToLower(strings.TrimSpace(jsonPkt.Params[i]))

				if _, ok := tm_mnemonic_simulated_data_map[param]; !ok {
					fmt.Println("Requested param does not exists in database:", param)

				}

				if !paramExists(client.subscriptions, param) {
					client.subscriptions = append(client.subscriptions, param)
				}

			}

		}

	} else {
		fmt.Println("Invalid Action Requested from Client:", jsonPkt.Action)
	}
}

func getInjectedTC(r *redis.Client) map[string]map[string]string {
	var tc injectedTC
	injectedTCMap := make(map[string]map[string]string)
	injected_data := r.HGetAll(ctx, "INJECTED_TC_FOR_SIMULATOR_MAP").Val()
	for key, value := range injected_data {
		json.Unmarshal([]byte(value), &tc)
		injectedTCMap[key] = map[string]string{"cmd": tc.Cmd, "code": tc.Code, "full_code": tc.FullCode, "data_part": tc.DataPart, "status": tc.Status}
	}
	return injectedTCMap
}

func startTCPublisher() {

	r := redis.NewClient(&redis.Options{Addr: "localhost:6379", Password: "", DB: 0})
	defer r.Close()
	for {
		injectedTCMap := getInjectedTC(r)
		for _, tc := range injectedTCMap {
			//fmt.Println(injectedTCData)
			fmtJSON, _ := json.Marshal(tc)
			isPrblm := false
			for _, client := range TCClients {
				err := client.connection.WriteMessage(ws.TextMessage, fmtJSON)
				//fmt.Println(tc)
				if err != nil {
					isPrblm = true
					fmt.Println(err)
				}
			}
			if isPrblm {
				removeChain1Client()
			}
			time.Sleep(100 * time.Nanosecond)
		}
		time.Sleep(700 * time.Millisecond)
	}

}

func wsTCEndPoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
		return
	}
	client := &clientInfo{}
	client.connection = conn
	TCClients = append(TCClients, client)
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	fmt.Println("New client connected from", ip, "to TC")
	fmt.Println("No. of TC Clients: ", len(chain1Clients))
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			logger.Println(err)
			return
		}
	}
}

func removeTCClient() {
	for index, client := range TCClients {
		if err := client.connection.WriteMessage(ws.PingMessage, []byte{}); err != nil {
			fmt.Println(err)
			TCClients = append(TCClients[:index], TCClients[index+1:]...)
			removeTCClient()
			break
		}
	}
}

func paramExists(list []string, value string) bool {
	for _, a := range list {
		if a == value || a == "*" {
			return true
		}
	}
	return false
}

func getParamIndex(list []string, value string) int {
	for index, a := range list {
		if a == value {
			return index
		}
	}
	return -1
}

func isParamFound(clientList []*clientInfo, value string) bool {
	for _, client := range clientList {
		paramList := client.subscriptions

		for _, a := range paramList {
			if a == value {
				return true
			}
		}
	}
	return false
}

func mapkey(m map[string]string, value string) (key string, ok bool) {
	for k, v := range m {
		if v == value {
			key = k
			ok = true
			return
		}
	}
	return
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
      var sub_scribe = {"action":"subscribe"}
	   sub_scribe["params"]= ["FRAME_ID"];  
	   var data = {}
        ws = new WebSocket("ws://172.20.26.1:9050/ws");
        ws.onopen = function(evt) {
            ws.send(JSON.stringify(sub_scribe));
        }
        ws.onclose = function(evt) {
			console.log("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
			let val = JSON.parse(evt.data);
			data[val.param] = JSON.parse(evt.data);
			
        }
        ws.onerror = function(evt) {
            console.log("ERROR: " + evt.data);
        }
          

</script>
</head>
<body>
<h1>How to Run</h1>
<h2>To get random TM just run :simulator.exe</h2>
<h2>To get Injected TM run with cmd line arguments: simulator.exe -random=false</h2>
<h1>How to see simulated data in browser</h1>
<h2>Open Console....use "data" variable</h2>
<h1>Python code to inject telemetry</h1>
<div>
<h2>import redis,json</h2>
<h2>r = redis.StrictRedis(host='localhost', port=6379, db=0)</h2>
<h2>injected_tm_from_tc = [{"mnemonic": "twt-op-sw15-sts", "value": "pos3", "lower_limit": -1,"upper_limit": -1},
{"mnemonic": "bcn-m1-oppwr", "value": "2.23", "lower_limit": 0, "upper_limit": 150},
]</h2>
<h2>r.set("injected_tm_from_tc",json.dumps(injected_tm_from_tc))</h2>
</div>
</body>
</html>
`))
