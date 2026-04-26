package internal

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/mainframe/tm-system/internal/models"
)

// Writer writes telemetry data points to InfluxDB.
type Writer struct {
	client *influxdb3.Client
	logger *slog.Logger
}

// NewWriter creates a new InfluxDB point writer.
func NewWriter(client *influxdb3.Client, logger *slog.Logger) *Writer {
	return &Writer{
		client: client,
		logger: logger.With("component", "influx-writer"),
	}
}

// Write writes a single telemetry value to InfluxDB as a point in the "telemetry"
// measurement. Tags include mnemonic, subsystem, chain, and type. For ANALOG mnemonics
// the value is stored as a float64 field; for BINARY it is stored as a string field.
func (w *Writer) Write(ctx context.Context, mnem models.TmMnemonic, value string, ts time.Time) {
	id := string(mnem.ID)
	tags := map[string]string{
		"mnemonic":  id,
		"subsystem": mnem.Subsystem,
		"type":      mnem.Type,
	}

	fields := make(map[string]interface{})

	if mnem.IsAnalog() {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			w.logger.Warn("failed to parse analog value as float64",
				"mnemonic", id,
				"value", value,
				"error", err,
			)
			fields["value"] = value // store as string fallback
		} else {
			fields["value"] = f
		}
	} else {
		fields["value"] = value
	}

	point := influxdb3.NewPointWithMeasurement("telemetry").
		SetTimestamp(ts)

	for k, v := range tags {
		point = point.SetTag(k, v)
	}
	for k, v := range fields {
		switch val := v.(type) {
		case float64:
			point = point.SetDoubleField(k, val)
		case string:
			point = point.SetStringField(k, val)
		}
	}

	if err := w.client.WritePoints(ctx, []*influxdb3.Point{point}); err != nil {
		w.logger.Error("failed to write point to InfluxDB",
			"mnemonic", id,
			"error", err,
		)
		return
	}

	w.logger.Debug("point written", "mnemonic", id, "value", value)
}
