package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

const (
	managedFirewallRuleName = "huginn-managed"
	metadataBase            = "http://metadata.google.internal/computeMetadata/v1"
)

func EnsureFirewall(ctx context.Context, cfg Config) {
	log.Printf("huginn: firewall: check starting manage=%v provider=%q", cfg.FirewallManage, cfg.CloudProvider)
	if !cfg.FirewallManage {
		return
	}

	var err error
	switch cfg.CloudProvider {
	case "gcp":
		err = ensureGCPFirewall(ctx, cfg)
	default:
		log.Printf("huginn: firewall: no automation for cloud_provider=%q yet — open ports %s (tcp) / %s (udp) manually",
			cfg.CloudProvider, portNumber(cfg.HTTPListenAddr), portNumber(cfg.UDPListenAddr))
		return
	}

	if err == nil {
		return
	}

	if isPermissionDenied(err) {
		log.Printf(`huginn: firewall: missing permission to manage GCP firewall rules.
  Run this once (from Cloud Shell), then restart huginn:

  gcloud projects add-iam-policy-binding $(gcloud config get-value project) \
    --member="serviceAccount:$(gcloud compute instances describe $(hostname) \
      --zone=$(curl -s -H 'Metadata-Flavor: Google' \
      http://metadata.google.internal/computeMetadata/v1/instance/zone | cut -d/ -f4) \
      --format='value(serviceAccounts[0].email)')" \
    --role="roles/compute.securityAdmin"
`)
		return
	}

	log.Printf("huginn: firewall: could not configure automatically: %v — open ports manually for now", err)
}

func ensureGCPFirewall(ctx context.Context, cfg Config) error {
	project, err := gcpMetadata(ctx, "/project/project-id")
	if err != nil {
		return fmt.Errorf("project id: %w", err)
	}

	token, err := gcpAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("access token: %w", err)
	}

	gamePorts := fmt.Sprintf("%d-%d", cfg.GamePort, cfg.GamePort+cfg.MaxInstances-1)

	rule := map[string]any{
		"name":         managedFirewallRuleName,
		"direction":    "INGRESS",
		"sourceRanges": []string{"0.0.0.0/0"},
		"allowed": []map[string]any{
			{"IPProtocol": "tcp", "ports": []string{portNumber(cfg.HTTPListenAddr)}},
			{"IPProtocol": "udp", "ports": []string{gamePorts}},
		},
		"description": "Managed by Huginn — edit huginn config, not this rule",
	}

	base := fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls", project)

	getResp, err := gcpAPI(ctx, token, http.MethodGet, base+"/"+managedFirewallRuleName, nil)
	if err != nil {
		return err
	}
	defer getResp.Body.Close()

	if getResp.StatusCode == http.StatusNotFound {
		resp, err := gcpAPI(ctx, token, http.MethodPost, base, rule)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return apiError(resp)
		}
		log.Printf("huginn: firewall: created rule %q (tcp:%s udp:%s)", managedFirewallRuleName,
			portNumber(cfg.HTTPListenAddr), gamePorts)
		return nil
	}

	if getResp.StatusCode >= 300 {
		return apiError(getResp)
	}

	resp, err := gcpAPI(ctx, token, http.MethodPut, base+"/"+managedFirewallRuleName, rule)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return apiError(resp)
	}

	log.Printf("huginn: firewall: updated rule %q (tcp:%s udp:%s)", managedFirewallRuleName,
		portNumber(cfg.HTTPListenAddr), gamePorts)
	return nil
}

func gcpAPI(ctx context.Context, token, method, url string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return httpClient.Do(req)
}

func apiError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return &gcpAPIErr{status: resp.StatusCode, body: string(body)}
}

type gcpAPIErr struct {
	status int
	body   string
}

func (e *gcpAPIErr) Error() string {
	return fmt.Sprintf("gcp api: status %d: %s", e.status, e.body)
}

func isPermissionDenied(err error) bool {
	var apiErr *gcpAPIErr
	if errors.As(err, &apiErr) {
		return apiErr.status == http.StatusForbidden
	}
	return false
}

func gcpMetadata(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataBase+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

func gcpAccessToken(ctx context.Context) (string, error) {
	body, err := gcpMetadata(ctx, "/instance/service-accounts/default/token")
	if err != nil {
		return "", err
	}
	var tok tokenResponse
	if err := json.Unmarshal([]byte(body), &tok); err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}
	if tok.AccessToken == "" {
		return "", errors.New("empty access token from metadata server")
	}
	return tok.AccessToken, nil
}

func portNumber(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return port
}
