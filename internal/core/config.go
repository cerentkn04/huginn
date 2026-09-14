package core

import (
	"errors"
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
	"strings"
)

type Config struct {
	Game             string `yaml:"game" json:"game"`
	Image            string `yaml:"image" json:"image"`
	MinInstances     int    `yaml:"min_instances" json:"min_instances"`
	MaxInstances     int    `yaml:"max_instances" json:"max_instances"`
	BufferSize       int    `yaml:"buffer_size" json:"buffer_size"`
	MaxPlayers       int    `yaml:"max_players" json:"max_players"`
	GamePort         int    `yaml:"port" json:"port"`
	UDPListenAddr    string `yaml:"udp_listen_addr" json:"udp_listen_addr"`
	HTTPListenAddr   string `yaml:"http_listen_addr" json:"http_listen_addr"`
	HeartbeatTimeout int    `yaml:"heartbeat_timeout_seconds" json:"heartbeat_timeout_seconds"`
	Mode             string `yaml:"mode" json:"mode"`
	PublicHost       string `yaml:"public_host" json:"public_host"`
	CloudProvider    string `yaml:"cloud_provider" json:"cloud_provider"`

	path string // where this was loaded from; unexported so it never serializes
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}
	cfg := Config{
		UDPListenAddr:    "0.0.0.0:9000",
		HTTPListenAddr:   ":8080",
		HeartbeatTimeout: 15,
		CloudProvider:    "gcp",
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
	cfg.path = path
	return cfg, nil
}
func (c Config) validate() error {
	var errs []string

	if c.Game == "" {
		errs = append(errs, "game is required")
	}
	if c.Image == "" {
		errs = append(errs, "image is required")
	}
	if c.GamePort <= 0 || c.GamePort > 65535 {
		errs = append(errs, "port must be between 1 and 65535")
	}
	if c.MinInstances <= 0 {
		errs = append(errs, "min_instances must be > 0")
	}
	if c.MaxInstances <= 0 {
		errs = append(errs, "max_instances must be > 0")
	}
	if c.MaxInstances < c.MinInstances {
		errs = append(errs, "max_instances must be >= min_instances")
	}
	if c.HeartbeatTimeout <= 0 {
		errs = append(errs, "heartbeat_timeout_seconds must be > 0")
	}
	if c.MaxPlayers <= 0 {
		errs = append(errs, "max_players must be > 0")
	}
	if c.BufferSize <= 0 {
		errs = append(errs, "buffer_size must be > 0")
	}
	if c.CloudProvider != "gcp" && c.CloudProvider != "aws" && c.CloudProvider != "custom" {
		errs = append(errs, "cloud_provider must be one of: gcp, aws, custom")
	}
	if c.CloudProvider == "custom" && c.PublicHost == "" {
		errs = append(errs, "public_host is required when cloud_provider is custom")
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}
