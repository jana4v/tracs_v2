package shared

import (
	"context"
	"time"

	redis "github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type ReqMessage struct {
	Action    string   `json:"action"`
	ParamList []string `json:"params"`
}

type TelemetryData struct {
	Mnemonic       string
	ProcessedValue string
	RawValue       int64
	UpperLimit     float64
	LowerLimit     float64
	Timestamp      time.Time
}

type TmPacket struct {
	Paramid    string  `json:"param_id"`
	Param      string  `json:"param"`
	SourceInfo string  `json:"source_info"`
	RawCount   int64   `json:"raw_count"`
	ProcValue  string  `json:"proc_value"`
	TimeStamp  string  `json:"time_stamp"`
	UpperLimit float64 `json:"upper_limit"`
	LowerLimit float64 `json:"lower_limit"`
	ErrDesc    string  `json:"err_desc"`
}

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

type ScosPkt struct {
	ParamList []ScosParamInfo `json:"params"`
	Stream    string          `json:"stream"`
	SeqCount  string          `json:"seqcount"`
	Time      string          `json:"time_stamp"`
	Error     string          `json:"err_desc"`
}

type TcSentPkt struct {
	Cmd      string `json:"cmd"`
	Code     string `json:"code"`
	FullCode string `json:"full_code"`
	DataPart string `json:"data_part"`
	Status   string `json:"status"`
	Time     string `json:"time"`
}

type ScosParamInfo struct {
	Mnemonic string `json:"param"`
	Value    string `json:"value"`
}

type Smon1totmpkt struct {
	Param     string `json:"param"`
	ProcValue string `json:"proc_value"`
	ErrDesc   string `json:"err_desc"`
}

type RedisKeysStruct struct {
	TM1_HEART_BEAT                string
	TM2_HEART_BEAT                string
	SMON1_HEART_BEAT              string
	SMON2_HEART_BEAT              string
	ADC1_HEART_BEAT               string
	ADC2_HEART_BEAT               string
	PTM1_HEART_BEAT               string
	PTM2_HEART_BEAT               string
	DWELL1_HEART_BEAT             string
	DWELL2_HEART_BEAT             string
	TM1_PKT                       string
	TM1_PROCESSED                 string
	TM1_RAW                       string
	TM1_LOWER_LIMIT               string
	TM1_UPPER_LIMIT               string
	TM1_TIMESTAMP                 string
	TM1_SOURCE_INFO               string
	TM1_MNEMONIC                  string
	TM1_MAP                       string
	TM1_PARAM                     string
	TM2_PKT                       string
	TM2_PROCESSED                 string
	TM2_RAW                       string
	TM2_LOWER_LIMIT               string
	TM2_UPPER_LIMIT               string
	TM2_TIMESTAMP                 string
	TM2_SOURCE_INFO               string
	TM2_MNEMONIC                  string
	TM2_MAP                       string
	TM2_PARAM                     string
	SMON1_MAP                     string
	SMON2_MAP                     string
	ADC1_MAP                      string
	ADC2_MAP                      string
	TC_SENT                       string
	ANALOG_TM_DYNAMIC_LIMITS      string
	DIGITAL_TM_DYNAMIC_LIMITS     string
	TM1_ARRAY                     string
	TM2_ARRAY                     string
	TC_DERIVED_DATA_COMMANDS      string // This is the key for the hash that stores the mapping of TC data commands to data code descriptions if defined in database
	TC_DERIVED_DATA_COMMANDS_CODE string // This is the key for the hash that stores the mapping of TC data commands to data code
	BAS_CURRENT                   string
	BAS_DELTA_CURRENT             string
	DERIVED_TM                    string
	DERIVED_TM_KV                 string
	PTM1_MAP                      string
	PTM2_MAP                      string
	TC_FILES_STATUS               string
	TC_FILES_START_TIME           string
	TC_FILES_EXECUTION_TIME       string
	TC_FILES_WAIT_TIME            string
	REDIS_CHANNEL_TO_WAMP_PUBLISH string
	TM_MAP                        string
}

var RedisKeys = RedisKeysStruct{
	TM1_HEART_BEAT:                "TM1_HEART_BEAT",
	TM2_HEART_BEAT:                "TM2_HEART_BEAT",
	SMON1_HEART_BEAT:              "SMON1_HEART_BEAT",
	SMON2_HEART_BEAT:              "SMON2_HEART_BEAT",
	ADC1_HEART_BEAT:               "ADC1_HEART_BEAT",
	ADC2_HEART_BEAT:               "ADC2_HEART_BEAT",
	PTM1_HEART_BEAT:               "PTM1_HEART_BEAT",
	PTM2_HEART_BEAT:               "PTM2_HEART_BEAT",
	DWELL1_HEART_BEAT:             "DWELL1_HEART_BEAT",
	DWELL2_HEART_BEAT:             "DWELL2_HEART_BEAT",
	TM1_PKT:                       "TM1_PKT",
	TM1_MAP:                       "TM1_MAP",
	TM2_PKT:                       "TM2_PKT",
	TM2_MAP:                       "TM2_MAP",
	SMON1_MAP:                     "SMON1_MAP",
	SMON2_MAP:                     "SMON2_MAP",
	ADC1_MAP:                      "ADC1_MAP",
	ADC2_MAP:                      "ADC2_MAP",
	TC_SENT:                       "TC_SENT",
	ANALOG_TM_DYNAMIC_LIMITS:      "ANALOG_TM_DYNAMIC_LIMITS",
	DIGITAL_TM_DYNAMIC_LIMITS:     "DIGITAL_TM_DYNAMIC_LIMITS",
	TM1_ARRAY:                     "TM1_ARRAY",
	TM2_ARRAY:                     "TM2_ARRAY",
	TC_DERIVED_DATA_COMMANDS:      "TC_DERIVED_DATA_COMMANDS",
	TC_DERIVED_DATA_COMMANDS_CODE: "TC_DERIVED_DATA_COMMANDS_CODE",
	BAS_CURRENT:                   "BAS_CURRENT",
	BAS_DELTA_CURRENT:             "BAS_DELTA_CURRENT",
	DERIVED_TM:                    "DTM",
	DERIVED_TM_KV:                 "DTM_MAP",
	PTM1_MAP:                      "PTM1_MAP",
	PTM2_MAP:                      "PTM2_MAP",
	TC_FILES_STATUS:               "TC_FILES_STATUS",
	TC_FILES_START_TIME:           "TC_FILES_START_TIME",
	TC_FILES_EXECUTION_TIME:       "TC_FILES_EXECUTION_TIME",
	TC_FILES_WAIT_TIME:            "TC_FILES_WAIT_TIME",
	REDIS_CHANNEL_TO_WAMP_PUBLISH: "REDIS_CHANNEL_TO_WAMP_PUBLISH",
	TM_MAP:                        "TM_MAP",
}

type HeartBeat struct {
	CONNECTION_FAILED string
	CONNECTED         string
	DATA_BREAK        string
	OK                string
}

var HeartBeatStatus = HeartBeat{
	CONNECTION_FAILED: "CONNECTION_FAILED",
	CONNECTED:         "CONNECTED",
	DATA_BREAK:        "DATA_BREAK",
	OK:                "OK",
}

const TmRefreshRate = 1
const (
	Wamp_tc_sent            = "tc_sent"
	Wamp_tc_file_status     = "com.tc_file.status"
	Wamp_tm1                = "tm1"
	Wamp_tm2                = "tm2"
	Wamp_ptm1               = "ptm1"
	Wamp_ptm2               = "ptm2"
	Wamp_smon1              = "smon1"
	Wamp_smon2              = "smon2"
	Wamp_adc1               = "adc1"
	Wamp_adc2               = "adc2"
	Wamp_ud                 = "user_defined"
	Wamp_procees_status     = "process_status"
	Wamp_user_defined_tm    = "user_defined_tm"
	Wamp_user_defined_tm_kv = "user_defined_tm_kv"
)

const (
	TM1_PORT     = "9050"
	TM2_PORT     = "9051"
	SMON1_PORT   = "9060"
	ADC1_PORT    = "9011"
	SMON2_PORT   = "9061"
	ADC2_PORT    = "9063"
	PTM1_PORT    = "9052"
	PTM2_PORT    = "9053"
	TC_SENT_PORT = "9070"
)

// Request represents the structure of the incoming request
type TcRequest struct {
	Action       string `json:"action"`
	ProcName     string `json:"proc_name"`
	ProcSrc      string `json:"proc_src"`
	ProcMode     string `json:"proc_mode"`
	ProcPriority string `json:"proc_priority"`
}

// Response represents the structure of the outgoing response
type TcResponse struct {
	Ack       bool   `json:"ack"`
	ErrorMsg  string `json:"error_msg,omitempty"`
	ExeStatus string `json:"exe_status,omitempty"`
}

type RedisUmacsEnvData struct {
	UMACS_DATA_SERVER_IP      string
	UMACS_TC_IP               string
	UMACS_TC_PORT             string
	TC_API_URL                string
	TC_API_REQ_SOURCE         string
	TC_API_REQ_PRIORITY       string
	TC_API_REQ_EXECUTION_MODE string
	TC_API_REQ_SUBSYSTEM      string
}

func ReadUmacsEnvData(r *redis.Client) RedisUmacsEnvData {
	res := RedisUmacsEnvData{}
	SCC_DATA_SERVER_IP, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "UMACS_DATA_SERVER_IP").Result()
	if err != nil {
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "UMACS_DATA_SERVER_IP", "172.20.xx.xx").Result()
		res.UMACS_DATA_SERVER_IP = "172.20.xx.xx"
	} else {
		res.UMACS_DATA_SERVER_IP = SCC_DATA_SERVER_IP
	}
	UMACS_TC_IP, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "UMACS_TC_IP").Result()
	if err != nil {
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "UMACS_TC_IP", "172.20.xx.xx").Result()
		res.UMACS_TC_IP = "172.20.xx.xx"
	} else {
		res.UMACS_TC_IP = UMACS_TC_IP
	}
	UMACS_TC_PORT, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "UMACS_TC_PORT").Result()
	if err != nil {
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "UMACS_TC_PORT", "8787").Result()
		res.UMACS_TC_PORT = "8787"
	} else {
		res.UMACS_TC_PORT = UMACS_TC_PORT
	}

	TC_API_REQ_SOURCE, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "TC_API_REQ_SOURCE").Result()
	if err != nil {
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "TC_API_REQ_SOURCE", "pcc").Result()
		res.TC_API_REQ_SOURCE = "pcc"
	} else {
		res.TC_API_REQ_SOURCE = TC_API_REQ_SOURCE
	}

	TC_API_REQ_PRIORITY, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "TC_API_REQ_PRIORITY").Result()
	if err != nil {
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "TC_API_REQ_PRIORITY", "normal").Result()
		res.TC_API_REQ_PRIORITY = "normal"
	} else {
		res.TC_API_REQ_PRIORITY = TC_API_REQ_PRIORITY
	}

	TC_API_REQ_EXECUTION_MODE, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "TC_API_REQ_EXECUTION_MODE").Result()
	if err != nil {
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "TC_API_REQ_EXECUTION_MODE", "auto").Result()
		res.TC_API_REQ_EXECUTION_MODE = "auto"
	} else {
		res.TC_API_REQ_EXECUTION_MODE = TC_API_REQ_EXECUTION_MODE
	}

	TC_API_REQ_SUBSYSTEM, err := r.HGet(ctx, "ENV_VARIABLES_UMACS", "TC_API_REQ_SUBSYSTEM").Result()
	if err != nil {
		r.HSet(ctx, "ENV_VARIABLES_UMACS", "TC_API_REQ_SUBSYSTEM", "payload").Result()
		res.TC_API_REQ_SUBSYSTEM = "payload"
	} else {
		res.TC_API_REQ_SUBSYSTEM = TC_API_REQ_SUBSYSTEM
	}

	res.TC_API_URL = "http://" + res.UMACS_TC_IP + ":" + res.UMACS_TC_PORT + "/"
	return res
}
