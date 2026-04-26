package Telemetry

import (
	"context"
	"strconv"

	redis "github.com/go-redis/redis/v8"
)

func isFloat(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func stringToFloat(s string) (float64, bool) {
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func publish(ctx context.Context, rdb *redis.Client, channel string, message string) {
	err := rdb.Publish(ctx, channel, message).Err()
	if err != nil {
		logger.Println("Error publishing message:", err)
	}
}
