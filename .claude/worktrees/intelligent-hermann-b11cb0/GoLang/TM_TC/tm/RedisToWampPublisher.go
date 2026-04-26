package Telemetry

import (
	"context"
	"encoding/json"
	shared "scg/shared"

	"github.com/gammazero/nexus/v3/wamp"
)

type WampMessage struct {
	Topic string            `json:"topic"`
	Msg   map[string]string `json:"msg"`
}

type WampMessage1 struct {
	Topic string `json:"topic"`
	Msg   string `json:"msg"`
}

func RedisToWampMessagePublisher() {
	wampClient := shared.GetWampConnection()
	ctx := context.Background()
	channel := shared.RedisKeys.REDIS_CHANNEL_TO_WAMP_PUBLISH
	rdb := shared.GetRedisConnection()
	sub := rdb.Subscribe(ctx, channel)
	defer sub.Close()
	_, err := sub.Receive(ctx)
	if err != nil {
		logger.Println("Error subscribing to channel:", err)
		return
	}
	// Go channel which receives messages.
	ch := sub.Channel()

	for msg := range ch {
		logger.Println("Received message from", msg.Channel, ":", msg.Payload)

		var wampMsg interface{}

		// Try to unmarshal as WampMessage
		if err := json.Unmarshal([]byte(msg.Payload), &wampMsg); err == nil {
			if wm, ok := wampMsg.(WampMessage); ok {
				wampClient.Publish(wm.Topic, nil, wamp.List{wm.Msg}, nil)
				continue
			}
		}
		// Try to unmarshal as WampMessage1
		if err := json.Unmarshal([]byte(msg.Payload), &wampMsg); err == nil {
			if wm1, ok := wampMsg.(WampMessage1); ok {
				wampClient.Publish(wm1.Topic, nil, wamp.List{wm1.Msg}, nil)
				continue
			}
		}
		logger.Println("Received unknown message type")
	}
}
