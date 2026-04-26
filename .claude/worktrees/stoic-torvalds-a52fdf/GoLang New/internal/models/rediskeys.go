package models

import "fmt"

// Dynamic Redis key builders — replaces the old 143-field RedisKeysStruct.
// Keys are generated based on chain type and number, enabling configurable chain counts.

// ChainMapKey returns the Redis hash key for a chain's telemetry map.
// Example: ChainMapKey("TM", 1) → "TM1_MAP"
func ChainMapKey(chainType string, n int) string {
	return fmt.Sprintf("%s%d_MAP", chainType, n)
}

// ChainMapKeyByName returns the Redis hash key for a named chain.
// Example: ChainMapKeyByName("TM1") → "TM1_MAP"
func ChainMapKeyByName(chainName string) string {
	return chainName + "_MAP"
}

// ChainPktKey returns the Redis hash key for raw packet storage.
// Example: ChainPktKey("TM1") → "TM1_PKT"
func ChainPktKey(chainName string) string {
	return chainName + "_PKT"
}

// HeartbeatChannelKey returns the Redis pub/sub channel for chain heartbeats.
// Example: HeartbeatChannelKey("TM1") → "TM1_HEARTBEAT_CHANNEL"
func HeartbeatChannelKey(chainName string) string {
	return chainName + "_HEARTBEAT_CHANNEL"
}

// HeartbeatStatusKey returns the Redis key for chain heartbeat status value.
// Example: HeartbeatStatusKey("TM1") → "TM1_HEART_BEAT"
func HeartbeatStatusKey(chainName string) string {
	return chainName + "_HEART_BEAT"
}

// LastDataTimeKey returns the Redis key for tracking last data received time per chain.
// Example: LastDataTimeKey("TM1") → "TM1_LAST_DATA_TIME"
func LastDataTimeKey(chainName string) string {
	return chainName + "_LAST_DATA_TIME"
}

// Well-known Redis map and channel constants (SRS Section 17).
const (
	// Data Maps
	TMMap                      = "TM_MAP"
	UnifiedTMMap               = "UNIFIED_TM_MAP" // Combined map: all streams, id_mnemonic keys
	SimulatedTMMap             = "SIMULATED_TM_MAP"
	UDTMMap                    = "UDTM_MAP"
	DTMMap                     = "DTM_MAP"
	TMChainMismatchesMap       = "TM_CHAIN_MISMATCHES_MAP"
	TMLimitFailuresMap         = "TM_LIMIT_FAILURES_MAP"
	TMExpectedDigitalStatesMap = "TM_EXPECTED_DIGITAL_STATES_MAP"

	// Suppression Map (EXPECTED pre-declarations before SEND commands)
	TMLimitSuppressionMap = "TM_LIMIT_SUPPRESSION_MAP"

	// Configuration Maps
	TMSimulatorCfgMap = "TM_SIMULATOR_CFG_MAP"
	TMSoftwareCfgMap  = "TM_SOFTWARE_CFG_MAP"
	SoftwareCfgMap    = "SOFTWARE_CFG_MAP"

	// Pub/Sub Channels
	TMSimulatorChannel     = "TM_SIMULATOR_CHANNEL"
	TMSimulatorCtrlChannel = "TM_SIMULATOR_CTRL_CHANNEL"
	TMLimitChanged         = "TM_LIMIT_CHANGED"
	MdbTmMnemonicsUpdated  = "MDB_TM_MNEMONICS_UPDATED"
	MdbTcCommandsUpdated   = "MDB_TC_COMMANDS_UPDATED"
	DTMProceduresUpdated   = "DTM_PROCEDURES_UPDATED"

	// TC Command Priority Queue (hardware-mode SEND)
	// TC_COMMAND_QUEUE is a Redis sorted set; score = priority (lower = higher priority).
	// TC_COMMAND_COMPLETED is the per-request pub/sub channel prefix; append ":{request_id}".
	TCCommandQueueKey        = "TC_COMMAND_QUEUE"
	TCCommandCompletedPrefix = "TC_COMMAND_COMPLETED"
)

// TM_SIMULATOR_CFG_MAP keys (SRS Section 3.2).
const (
	SimCfgEnable      = "ENABLE"
	SimCfgSampleDelay = "SAMPLE_DELAY"
	SimCfgMode        = "MODE"
)

// Simulator modes (SRS Section 3.3).
const (
	SimModeRandom = "RANDOM"
	SimModeFixed  = "FIXED"
)

// TM_SOFTWARE_CFG_MAP keys (SRS Section 14.2).
const (
	CfgChainCompareMode         = "CHAIN_COMPARE_MODE"
	CfgChainCompareDelaySeconds = "CHAIN_COMPARE_DELAY_SECONDS"
	CfgMasterFrameWaitCount     = "MASTER_FRAME_WAIT_COUNT"
	CfgTMDataTimeoutSeconds     = "TM_DATA_TIMEOUT_SECONDS"
	CfgSMONDataTimeoutSeconds   = "SMON_DATA_TIMEOUT_SECONDS"
	CfgADCDataTimeoutSeconds    = "ADC_DATA_TIMEOUT_SECONDS"
	CfgTMStorageEnable          = "TM_STORAGE_ENABLE"
	CfgInfluxLoggingEnable      = "INFLUXDB_LOGGING_ENABLE"

	// EXPECTED suppression window config (SRS Section 17.3).
	CfgMasterFrameDurationMs   = "MASTER_FRAME_DURATION_MS"
	CfgSuppressionWindowFrames = "SUPPRESSION_WINDOW_FRAMES"
)

// Chain comparison modes (SRS Section 6.3).
const (
	CompareModeA = "A" // Frame ID Sync Mode
	CompareModeB = "B" // Time Delta Mode
)
