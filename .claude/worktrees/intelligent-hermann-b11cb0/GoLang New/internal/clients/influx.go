package clients

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// NewInfluxClient creates an InfluxDB 3 OSS client with exponential backoff retry.
// Uses the influxdb3-go client for InfluxDB 3 OSS as specified in SRS Section 13.
func NewInfluxClient(ctx context.Context, url, token, database string, logger *slog.Logger) (*influxdb3.Client, error) {
	backoff := time.Second
	maxBackoff := 30 * time.Second
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		client, err := influxdb3.New(influxdb3.ClientConfig{
			Host:     url,
			Token:    token,
			Database: database,
		})
		if err == nil {
			logger.Info("connected to InfluxDB 3", "url", url, "database", database)
			return client, nil
		}

		logger.Warn("InfluxDB connection failed, retrying",
			"url", url,
			"attempt", i+1,
			"backoff", backoff,
			"error", err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, maxBackoff)
	}

	return nil, fmt.Errorf("failed to connect to InfluxDB at %s after %d retries", url, maxRetries)
}
