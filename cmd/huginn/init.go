package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func runInit() {
	reader := bufio.NewReader(os.Stdin)

	game := prompt(reader, "Game name", "my-game")
	image := prompt(reader, "Docker image (e.g. my-server:latest)", "")
	minInstances := promptInt(reader, "Minimum instances", 2)
	maxInstances := promptInt(reader, "Maximum instances", 10)
	maxPlayers := promptInt(reader, "Max players per instance", 16)
	port := promptInt(reader, "Game server UDP port", 7778)
	token := generateToken()

	cloudProvider, gcpProject, gcpZone, publicHost, firewallManage := promptCloudSetup(reader)

	cfg := fmt.Sprintf(`game: %s
image: %s
min_instances: %d
max_instances: %d
buffer_size: 3
max_players: %d
port: %d
heartbeat_timeout_seconds: 15
udp_listen_addr: 0.0.0.0:9000
http_listen_addr: :8080
mode: sdk
cloud_provider: %s
gcp_project: %s
gcp_zone: %s
public_host: %s
firewall_manage: %t
host_auto_scaling_enabled: false
auth_token: %s
`, game, image, minInstances, maxInstances, maxPlayers, port, cloudProvider, gcpProject, gcpZone, publicHost, firewallManage, token)

	if err := os.WriteFile("config.yaml", []byte(cfg), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: failed to write config.yaml: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wrote config.yaml — run `huginn start config.yaml` to launch your fleet.")
}

func promptCloudSetup(reader *bufio.Reader) (provider, project, zone, publicHost string, firewallManage bool) {
	enableGCP := promptYesNo(reader, "Enable GCP features (host provisioning, firewall management)?", false)
	if !enableGCP {
		publicHost = prompt(reader, "Public host/IP for this machine (required without GCP)", "")
		for publicHost == "" {
			fmt.Println("public_host is required when GCP features are disabled")
			publicHost = prompt(reader, "Public host/IP for this machine", "")
		}
		return "custom", "", "", publicHost, false
	}

	for {
		project = prompt(reader, "GCP project ID", "")
		if isValidGCPProjectID(project) {
			break
		}
		fmt.Println("invalid project ID — must be 6-30 lowercase letters, digits, or hyphens, starting with a letter")
	}
	for {
		zone = prompt(reader, "GCP zone (e.g. us-central1-a)", "us-central1-a")
		if isValidGCPZone(zone) {
			break
		}
		fmt.Println("invalid zone format — expected something like us-central1-a")
	}

	fmt.Println("Note: this only validates the format of what you entered — it does not check GCP credentials or create anything. Host auto-scaling stays OFF by default; enable it later in the Config UI when you're ready.")
	return "gcp", project, zone, "", true
}

func promptYesNo(reader *bufio.Reader, label string, def bool) bool {
	defStr := "y/N"
	if def {
		defStr = "Y/n"
	}
	fmt.Printf("%s [%s]: ", label, defStr)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}

func isValidGCPProjectID(id string) bool {
	if len(id) < 6 || len(id) > 30 {
		return false
	}
	if id[0] < 'a' || id[0] > 'z' {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '-' {
			return false
		}
	}
	return true
}

func isValidGCPZone(zone string) bool {
	parts := strings.Split(zone, "-")
	if len(parts) != 3 {
		return false
	}
	last := parts[2]
	return len(last) == 1 && last[0] >= 'a' && last[0] <= 'z'
}

func prompt(reader *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func promptInt(reader *bufio.Reader, label string, def int) int {
	for {
		s := prompt(reader, label, fmt.Sprintf("%d", def))
		n, err := strconv.Atoi(s)
		if err != nil {
			fmt.Println("please enter a number")
			continue
		}
		return n
	}
}
