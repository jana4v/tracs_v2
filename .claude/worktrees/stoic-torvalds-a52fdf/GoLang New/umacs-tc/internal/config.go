package internal

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type UmacsEnvData struct {
	TcIP                string
	TcPort              string
	DataServerIP        string
	APIReqSource        string
	APIReqPriority      string
	APIReqExecutionMode string
	APIReqSubsystem     string
}

func NewUmacsEnvData(rdb *redis.Client, cfg *struct {
	TcIP                 string `yaml:"tc_ip"`
	TcPort               string `yaml:"tc_port"`
	DataServerIP         string `yaml:"data_server_ip"`
	APIReqSource         string `yaml:"api_req_source"`
	APIReqPriority       string `yaml:"api_req_priority"`
	APIReqExecutionMode  string `yaml:"api_req_execution_mode"`
	APIReqSubsystem      string `yaml:"api_req_subsystem"`
}, logger *slog.Logger) *UmacsEnvData {
	ctx := context.Background()
	env := &UmacsEnvData{}

	env.TcIP, _ = rdb.HGet(ctx, envVariablesUmacsKey, "UMACS_TC_IP").Result()
	if env.TcIP == "" {
		env.TcIP = cfg.TcIP
		rdb.HSet(ctx, envVariablesUmacsKey, "UMACS_TC_IP", env.TcIP)
		logger.Info("set default UMACS_TC_IP", "value", env.TcIP)
	}

	env.TcPort, _ = rdb.HGet(ctx, envVariablesUmacsKey, "UMACS_TC_PORT").Result()
	if env.TcPort == "" {
		env.TcPort = cfg.TcPort
		rdb.HSet(ctx, envVariablesUmacsKey, "UMACS_TC_PORT", env.TcPort)
		logger.Info("set default UMACS_TC_PORT", "value", env.TcPort)
	}

	env.DataServerIP, _ = rdb.HGet(ctx, envVariablesUmacsKey, "UMACS_DATA_SERVER_IP").Result()
	if env.DataServerIP == "" {
		env.DataServerIP = cfg.DataServerIP
		rdb.HSet(ctx, envVariablesUmacsKey, "UMACS_DATA_SERVER_IP", env.DataServerIP)
		logger.Info("set default UMACS_DATA_SERVER_IP", "value", env.DataServerIP)
	}

	env.APIReqSource, _ = rdb.HGet(ctx, envVariablesUmacsKey, "TC_API_REQ_SOURCE").Result()
	if env.APIReqSource == "" {
		env.APIReqSource = cfg.APIReqSource
		rdb.HSet(ctx, envVariablesUmacsKey, "TC_API_REQ_SOURCE", env.APIReqSource)
		logger.Info("set default TC_API_REQ_SOURCE", "value", env.APIReqSource)
	}

	env.APIReqPriority, _ = rdb.HGet(ctx, envVariablesUmacsKey, "TC_API_REQ_PRIORITY").Result()
	if env.APIReqPriority == "" {
		env.APIReqPriority = cfg.APIReqPriority
		rdb.HSet(ctx, envVariablesUmacsKey, "TC_API_REQ_PRIORITY", env.APIReqPriority)
		logger.Info("set default TC_API_REQ_PRIORITY", "value", env.APIReqPriority)
	}

	env.APIReqExecutionMode, _ = rdb.HGet(ctx, envVariablesUmacsKey, "TC_API_REQ_EXECUTION_MODE").Result()
	if env.APIReqExecutionMode == "" {
		env.APIReqExecutionMode = cfg.APIReqExecutionMode
		rdb.HSet(ctx, envVariablesUmacsKey, "TC_API_REQ_EXECUTION_MODE", env.APIReqExecutionMode)
		logger.Info("set default TC_API_REQ_EXECUTION_MODE", "value", env.APIReqExecutionMode)
	}

	env.APIReqSubsystem, _ = rdb.HGet(ctx, envVariablesUmacsKey, "TC_API_REQ_SUBSYSTEM").Result()
	if env.APIReqSubsystem == "" {
		env.APIReqSubsystem = cfg.APIReqSubsystem
		rdb.HSet(ctx, envVariablesUmacsKey, "TC_API_REQ_SUBSYSTEM", env.APIReqSubsystem)
		logger.Info("set default TC_API_REQ_SUBSYSTEM", "value", env.APIReqSubsystem)
	}

	logger.Info("UMACS environment loaded", 
		"TC_IP", env.TcIP,
		"TC_PORT", env.TcPort,
		"API_REQ_SOURCE", env.APIReqSource,
		"API_REQ_PRIORITY", env.APIReqPriority,
		"API_REQ_EXECUTION_MODE", env.APIReqExecutionMode,
	)

	return env
}
