package core

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func DiscoverPublicHost(cfg Config) string {
	if cfg.PublicHost != "" {
		return cfg.PublicHost
	}

	url := "http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip"

	client := http.Client{Timeout: 2 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("huginn: could not build metadata request: %v", err)
		return ""
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("huginn: could not reach GCP metadata server (not on GCP?): %v", err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("unhealthy connection")
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("huginn: could not read metadata response: %v", err)
		return ""
	}

	return strings.TrimSpace(string(body))
}
