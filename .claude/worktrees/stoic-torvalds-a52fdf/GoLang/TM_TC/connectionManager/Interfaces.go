package connectionManager

import (
	"context"
	"net/url"
	"time"

	"github.com/gammazero/nexus/v3/client"
	redis "github.com/go-redis/redis/v8"
	ws "github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"scg/logs"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxApi "github.com/influxdata/influxdb-client-go/v2/api"
)

var WebSocketsRetryTime_Seconds time.Duration = 10 * time.Second

func GetRedisConnection() *redis.Client {
	r := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err := r.Ping(context.Background()).Result()
	if err != nil {
		logs.Logger.Println("Failed to Connect Redis Server")
		logs.Logger.Println(err)
	}
	return r
}

var cfg = client.Config{
	Realm:  realm,
	Logger: logs.Logger,
}

const (
	addr  = "ws://localhost:8086"
	realm = "realm1"
	tm1   = "com.GRCD.TM"
)

func GetWampConnection() *client.Client {
	wamp, err := client.ConnectNet(context.Background(), addr, cfg)
	if err != nil {
		logs.Logger.Println(err)
	}
	return wamp
}

func GetMongoConnection() (*mongo.Client, error) {

	ctx := context.TODO()
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	return client, err
}

func GetWebsocketConnection(ipPort string) (*ws.Conn, error) {
	customURL := url.URL{Scheme: "ws", Host: ipPort, Path: "/ws"}
	conn, _, err := ws.DefaultDialer.Dial(customURL.String(), nil)
	return conn, err
}

func GetInfluxConnection() influxdb2.Client {
	//token := os.Getenv("INFLUX_TOKEN")
	token := "re-JJKuE7jvrTM65GNeESGQbBtn2oe9eSvJbCx8RJyCRNbJHb4C86yx3WLG-X1SmbI9pD_hZ_wf-lA3JqM1eUg=="
	url := "http://127.0.0.1:8086"
	//client := influxdb2.NewClient(url, token)
	client := influxdb2.NewClientWithOptions(url, token, influxdb2.DefaultOptions().SetBatchSize(1000).SetFlushInterval(1000))
	return client
}

func GetInfluxWriteAPI(bucket_name string) influxApi.WriteAPI {
	client := GetInfluxConnection()
	writeAPI := client.WriteAPI("ISRO", bucket_name)
	return writeAPI
}

// func WriteScTelemetryDataToInfuxDB(write_api influxApi.WriteAPI, tm_chain string, data TmPacket) {
// 	tags := map[string]string{
// 		"source":   "satellite",
// 		"chain":    tm_chain,
// 		"mnemonic": data.Param,
// 	}
// 	fields := map[string]interface{}{
// 		"processed_value": data.ProcValue,
// 		"raw_count":       data.RawCount,
// 		"upper_limit":     data.UpperLimit,
// 		"lower_limit":     data.LowerLimit,
// 	}
// 	timestamp, _ := time.Parse(time.RFC3339, data.TimeStamp)
// 	point := influxdb2.NewPoint("measurement1", tags, fields, timestamp)
// 	write_api.WritePoint(point)
// }
