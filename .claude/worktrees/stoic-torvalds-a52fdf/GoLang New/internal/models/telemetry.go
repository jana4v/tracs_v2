package models

// TmPacket represents a single telemetry parameter received via WebSocket from the data server.
// Preserved from old shared.TmPacket with identical JSON tags for wire compatibility.
type TmPacket struct {
	ParamID    string  `json:"param_id"`
	Param      string  `json:"param"`
	SourceInfo string  `json:"source_info"`
	RawCount   int64   `json:"raw_count"`
	ProcValue  string  `json:"proc_value"`
	TimeStamp  string  `json:"time_stamp"`
	UpperLimit float64 `json:"upper_limit"`
	LowerLimit float64 `json:"lower_limit"`
	ErrDesc    string  `json:"err_desc"`
}

// PTmPacket represents a batch of processed telemetry parameters in a single frame.
// Preserved from old shared.PTmPacket.
type PTmPacket struct {
	Param      []string  `json:"params"`
	SourceInfo string    `json:"source_info"`
	RawCount   []int64   `json:"raw_counts"`
	ProcValue  []string  `json:"proc_values"`
	TimeStamp  string    `json:"time_stamp"`
	UpperLimit []float64 `json:"upper_limits"`
	LowerLimit []float64 `json:"lower_limits"`
	ErrDesc    string    `json:"err_desc"`
}

// ScosPkt represents a SCOS/SMON/ADC telemetry frame.
// Preserved from old shared.ScosPkt.
type ScosPkt struct {
	ParamList []ScosParamInfo `json:"params"`
	Stream    string          `json:"stream"`
	SeqCount  string          `json:"seqcount"`
	Time      string          `json:"time_stamp"`
	Error     string          `json:"err_desc"`
}

// ScosParamInfo represents a single parameter within a ScosPkt.
type ScosParamInfo struct {
	Mnemonic string `json:"param"`
	Value    string `json:"value"`
}

// ReqMessage is the WebSocket subscription request sent to the data server on connect.
type ReqMessage struct {
	Action    string   `json:"action"`
	ParamList []string `json:"params"`
}

// HeartbeatPayload is the heartbeat message published to Redis channels (SRS Section 12.5).
type HeartbeatPayload struct {
	Chain      string `json:"chain"`
	Status     string `json:"status"`
	LastDataTs string `json:"lastDataTs"`
	Timestamp  string `json:"timestamp"`
}

// IDMnemonic combines a parameter ID and mnemonic into the canonical
// id_mnemonic key format used in UNIFIED_TM_MAP and NATS payloads.
//
//	IDMnemonic("12345", "cpu_temp") → "12345_cpu_temp"
//	IDMnemonic("",      "cpu_temp") → "cpu_temp"
func IDMnemonic(paramID, mnemonic string) string {
	if paramID == "" {
		return mnemonic
	}
	return paramID + "_" + mnemonic
}

// HeartbeatStatus constants for chain liveness tracking.
const (
	StatusConnectionFailed = "CONNECTION_FAILED"
	StatusConnected        = "CONNECTED"
	StatusOK               = "OK"
	StatusDataBreak        = "DATA_BREAK"
	StatusActive           = "ACTIVE"
	StatusInactive         = "INACTIVE"
)
