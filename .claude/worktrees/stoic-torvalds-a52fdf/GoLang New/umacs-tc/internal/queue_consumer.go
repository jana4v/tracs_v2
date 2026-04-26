package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	TCCommandQueueKey        = "TC_COMMAND_QUEUE"
	TCCommandCompletedPrefix = "TC_COMMAND_COMPLETED"
)

// TCCommandPayload is the schema written by Julia's enqueue_command().
type TCCommandPayload struct {
	RequestID   string                 `json:"request_id"`
	ProcedureID string                 `json:"procedure_id"`
	Priority    int                    `json:"priority"`
	Procedure   string                 `json:"procedure,omitempty"`
	Command     string                 `json:"command,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Timestamp   string                 `json:"timestamp,omitempty"`
}

// TCCommandResult is published to "TC_COMMAND_COMPLETED:{request_id}" on completion.
type TCCommandResult struct {
	RequestID   string `json:"request_id"`
	ProcedureID string `json:"procedure_id"`
	Status      string `json:"status"`
	ErrorMsg    string `json:"error_msg,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

// QueueConsumer drains TC_COMMAND_QUEUE one item at a time and notifies
// the originating Julia task via a per-request Redis pub/sub channel.
// Only a single goroutine should call Run — this serialises dispatch to
// the UMACS TC API which is inherently sequential.
type QueueConsumer struct {
	rdb     *redis.Client
	handler *Handler
	logger  *slog.Logger
}

func NewQueueConsumer(rdb *redis.Client, h *Handler, l *slog.Logger) *QueueConsumer {
	return &QueueConsumer{
		rdb:     rdb,
		handler: h,
		logger:  l.With("component", "queue-consumer"),
	}
}

// Run blocks until ctx is cancelled, processing one command at a time.
func (qc *QueueConsumer) Run(ctx context.Context) {
	qc.logger.Info("TC command queue consumer started", "queue", TCCommandQueueKey)

	for {
		select {
		case <-ctx.Done():
			qc.logger.Info("queue consumer stopping")
			return
		default:
		}

		// BZPopMin blocks up to 5 s then loops, so ctx cancellation is checked promptly.
		z, err := qc.rdb.BZPopMin(ctx, 5*time.Second, TCCommandQueueKey).Result()
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				return
			}
			if err == redis.Nil {
				continue // timeout with no item — loop back
			}
			qc.logger.Error("BZPopMin error", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		raw, ok := z.Member.(string)
		if !ok {
			qc.logger.Error("unexpected member type in queue", "type", fmt.Sprintf("%T", z.Member))
			continue
		}

		var cmd TCCommandPayload
		if err := json.Unmarshal([]byte(raw), &cmd); err != nil {
			qc.logger.Error("failed to parse TC command payload", "raw", raw, "error", err)
			// The Julia caller will timeout — no notification possible without request_id
			continue
		}

		displayProcedure := cmd.Procedure
		if displayProcedure == "" {
			displayProcedure = cmd.Command
		}
		qc.logger.Info("dequeued TC command",
			"request_id", cmd.RequestID,
			"procedure_id", cmd.ProcedureID,
			"procedure", displayProcedure,
			"priority", cmd.Priority,
		)

		qc.dispatchAndNotify(ctx, cmd)
	}
}

// dispatchAndNotify sends the command to the UMACS TC API via the existing
// triggerFileWaitForExecutionComplete path, then publishes the result to the
// per-request completion channel so the Julia SEND statement can unblock.
func (qc *QueueConsumer) dispatchAndNotify(ctx context.Context, cmd TCCommandPayload) {
	result := TCCommandResult{
		RequestID:   cmd.RequestID,
		ProcedureID: cmd.ProcedureID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	procName := cmd.ProcedureID
	if procName == "" {
		if v, ok := cmd.Payload["proc_name"].(string); ok {
			procName = v
		} else if v, ok := cmd.Payload["procName"].(string); ok {
			procName = v
		}
	}
	procedure := cmd.Procedure
	if procedure == "" {
		procedure = cmd.Command
	}
	if procedure == "" {
		if v, ok := cmd.Payload["procedure"].(string); ok {
			procedure = v
		}
	}
	if procName == "" || procedure == "" {
		result.Status = "failed"
		result.ErrorMsg = "procedure_id and procedure are required"
		b, err := json.Marshal(result)
		if err != nil {
			qc.logger.Error("failed to marshal completion result", "error", err)
			return
		}
		channel := fmt.Sprintf("%s:%s", TCCommandCompletedPrefix, cmd.RequestID)
		if err := qc.rdb.Publish(ctx, channel, string(b)).Err(); err != nil {
			qc.logger.Error("failed to publish completion notification",
				"channel", channel,
				"error", err,
			)
		}
		return
	}

	_, err := qc.handler.transferFileTriggerExecutionAndWaitForCompletionInternal(procName, procedure)
	if err != nil {
		result.Status = "failed"
		result.ErrorMsg = err.Error()
		qc.logger.Error("TC command execution failed",
			"request_id", cmd.RequestID,
			"procedure_id", cmd.ProcedureID,
			"error", err,
		)
	} else {
		result.Status = "completed"
		qc.logger.Info("TC command completed",
			"request_id", cmd.RequestID,
			"command", cmd.Command,
		)
	}

	b, err := json.Marshal(result)
	if err != nil {
		qc.logger.Error("failed to marshal completion result", "error", err)
		return
	}

	channel := fmt.Sprintf("%s:%s", TCCommandCompletedPrefix, cmd.RequestID)
	if err := qc.rdb.Publish(ctx, channel, string(b)).Err(); err != nil {
		qc.logger.Error("failed to publish completion notification",
			"channel", channel,
			"error", err,
		)
	}
}
