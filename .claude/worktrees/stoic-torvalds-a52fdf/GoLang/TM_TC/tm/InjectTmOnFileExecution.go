package Telemetry

import (
	"context"
	shared "scg/shared"
)

func InjectTm() {
	ctx := context.Background()
	channel := "TC_FILE_EXECUTION_COMPLETED"
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

	// Consume messages.
	for msg := range ch {
		logger.Println("Received message from", msg.Channel, ":", msg.Payload)
		stored_data := rdb.HGetAll(ctx, msg.Payload).Val()
		for key, value := range stored_data {
			rdb.HSet(ctx, shared.RedisKeys.DERIVED_TM_KV, key, value).Result()
		}
		rdb.Del(ctx, msg.Payload).Err()
	}
}
