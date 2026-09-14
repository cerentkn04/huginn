package sidecar

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	InstanceID        string        // unique id for this game server instance
	Mode              string        // "log_parse" (Day 4) or "sdk" (Day 7)
	HTTPAddr          string        // address the sidecar's local HTTP server listens on (SDK mode)
	CoreUDPAddr       string        // Huginn Core's UDP listener address, e.g. "core:9000"
	LogPath           string        // (log_parse mode) path to the game server's log file to tail
	HeartbeatInterval time.Duration // how often to send a heartbeat regardless of change
}

func LoadConfig() Config {
	return Config{
		InstanceID:        getEnv("SIDECAR_INSTANCE_ID", "unknown"),
		Mode:              getEnv("SIDECAR_MODE", "log_parse"),
		HTTPAddr:          getEnv("SIDECAR_HTTP_ADDR", ":8090"),
		CoreUDPAddr:       getEnv("SIDECAR_CORE_UDP_ADDR", "127.0.0.1:9000"),
		LogPath:           getEnv("SIDECAR_LOG_PATH", "./server.log"),
		HeartbeatInterval: getEnvDuration("SIDECAR_HEARTBEAT_INTERVAL", 2*time.Second),
	}
}
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return fallback
}
