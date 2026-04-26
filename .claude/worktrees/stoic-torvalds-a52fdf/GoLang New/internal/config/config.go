package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// BaseConfig holds common configuration shared across all services.
type BaseConfig struct {
	Service ServiceConfig `mapstructure:"service"`
	Redis   RedisConfig   `mapstructure:"redis"`
	SQLite  SQLiteConfig  `mapstructure:"sqlite"`
}

// ServiceConfig identifies the service.
type ServiceConfig struct {
	Name     string `mapstructure:"name"`
	LogLevel string `mapstructure:"log_level"`
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// SQLiteConfig holds SQLite connection parameters.
type SQLiteConfig struct {
	Path     string `mapstructure:"path"`      // default: "./astra.db"
	InMemory bool   `mapstructure:"in_memory"` // use ":memory:" (testing)
}

// NATSConfig holds NATS connection and publishing parameters.
type NATSConfig struct {
	URL               string `mapstructure:"url"`                  // e.g. "nats://localhost:4222"
	Name              string `mapstructure:"name"`                 // e.g. "tm-ingest"
	SubjectPrefix     string `mapstructure:"subject_prefix"`       // e.g. "tm"
	PollIntervalMs    int    `mapstructure:"poll_interval_ms"`     // default 500
	SnapshotIntervalS int    `mapstructure:"snapshot_interval_s"`  // default 30
}

// InfluxConfig holds InfluxDB 3 OSS connection parameters.
type InfluxConfig struct {
	URL      string `mapstructure:"url"`
	Token    string `mapstructure:"token"`
	Org      string `mapstructure:"org"`
	Database string `mapstructure:"database"`
}

// ChainConfig defines a single telemetry chain connection.
type ChainConfig struct {
	Name string `mapstructure:"name"` // e.g. "TM1", "SMON2"
	Type string `mapstructure:"type"` // "TM", "SCOS" (for SMON/ADC)
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// WebSocketConfig holds WebSocket reconnection parameters.
type WebSocketConfig struct {
	RetryInitial    string `mapstructure:"retry_initial"`
	RetryMax        string `mapstructure:"retry_max"`
	RetryMultiplier int    `mapstructure:"retry_multiplier"`
}

// Load reads a YAML config file into the given struct.
func Load(configPath string, out interface{}) error {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Defaults
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("sqlite.path", "./astra.db")
	v.SetDefault("sqlite.in_memory", false)
	v.SetDefault("service.log_level", "info")
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.name", "tm-ingest")
	v.SetDefault("nats.subject_prefix", "tm")
	v.SetDefault("nats.poll_interval_ms", 800)
	v.SetDefault("nats.snapshot_interval_s", 30)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("read config %s: %w", configPath, err)
	}
	if err := v.Unmarshal(out); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}
