package core

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"github.com/goccy/go-yaml"
)

type Config struct {
	Game             string `yaml:"game"`
	Image            string `yaml:"image"`
	MinInstances     int    `yaml:"min_instances"`
	MaxInstances     int    `yaml:"max_instances"`
	BufferSize 	 int	`yaml:"buffer_size"`
	MaxPlayers	 int	`yaml:"max_players"`
	GamePort         int    `yaml:"port"`
	UDPListenAddr    string `yaml:"udp_listen_addr"`
	HTTPListenAddr   string `yaml:"http_listen_addr"`
	HeartbeatTimeout int    `yaml:"heartbeat_timeout_seconds"`
	Mode		 string  `yaml:"mode"`
	PublicHost	 string  `yaml:"public_host"`
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
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}
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
		if c.MaxPlayers <=0 {
		errs = append(errs, "max_players must be > 0")
	}
	if c.BufferSize <= 0 {
 	   errs = append(errs, "buffer_size must be > 0")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}
