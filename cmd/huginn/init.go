package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	clientToken := generateToken()

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
client_auth_token: %s
`, game, image, minInstances, maxInstances, maxPlayers, port, cloudProvider, gcpProject, gcpZone, publicHost, firewallManage, token, clientToken)

	if err := os.WriteFile("config.yaml", []byte(cfg), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: failed to write config.yaml: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wrote config.yaml — run `huginn start config.yaml` to launch your fleet.")
	fmt.Println()
	fmt.Println("Two tokens were generated:")
	fmt.Printf("  auth_token         (admin — dashboard login, full control): %s\n", token)
	fmt.Printf("  client_auth_token  (players — server discovery only):       %s\n", clientToken)
	fmt.Println("Put ONLY client_auth_token in huginn.json for your game client. Never ship auth_token to players.")

	writeSystemdUnit(reader, game)
}
func promptCloudSetup(reader *bufio.Reader) (provider, project, zone, publicHost string, firewallManage bool) {
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
	fmt.Println("Project ID and zone were only format-checked. The next step is optional and makes real changes to your GCP project.")
	promptGCPAutomatedSetup(reader, project)
	fmt.Println("Host auto-scaling stays OFF by default; enable it later in the Config UI.")
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
func writeSystemdUnit(reader *bufio.Reader, game string) {
	if promptYesNo(reader, "Generate a systemd service file for this fleet?", true) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "huginn: could not determine working directory: %v\n", err)
			return
		}
		binPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "huginn: could not locate huginn binary: %v\n", err)
			return
		}
		binPath, _ = filepath.EvalSymlinks(binPath)
		configPath := filepath.Join(cwd, "config.yaml")
		user := os.Getenv("USER")
		if user == "" {
			user = os.Getenv("LOGNAME")
		}
		if user == "" {
			fmt.Println("Could not determine current user — skipping systemd unit generation. You can create it manually.")
			return
		}

		unit := fmt.Sprintf(`[Unit]
Description=Huginn fleet manager (%s)
After=docker.service
Requires=docker.service

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s start %s
Restart=on-failure
RestartSec=5
User=%s

[Install]
WantedBy=multi-user.target
`, game, cwd, binPath, configPath, user)

		unitPath := filepath.Join(cwd, "huginn.service")
		if err := os.WriteFile(unitPath, []byte(unit), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "huginn: failed to write huginn.service: %v\n", err)
			return
		}
		fmt.Printf("Wrote %s\n", unitPath)

		if promptYesNo(reader, "Install and enable it now via sudo? (requires sudo privileges)", false) {
			if _, err := os.Stat("/etc/systemd/system/huginn.service"); err == nil {
				fmt.Println("Warning: /etc/systemd/system/huginn.service already exists and will be replaced.")
				if !promptYesNo(reader, "Overwrite it?", false) {
					fmt.Println("Skipped install.")
					return
				}
			}
			installSystemdUnit(unitPath)
		} else {
			fmt.Printf(`To install it later, run:
  sudo cp %s /etc/systemd/system/huginn.service
  sudo systemctl daemon-reload
  sudo systemctl enable --now huginn
`, unitPath)
		}
	}
}

func installSystemdUnit(unitPath string) {
	dest := "/etc/systemd/system/huginn.service"
	cp := exec.Command("sudo", "cp", unitPath, dest)
	cp.Stdout = os.Stdout
	cp.Stderr = os.Stderr
	if err := cp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: failed to install unit file: %v\n", err)
		return
	}

	reload := exec.Command("sudo", "systemctl", "daemon-reload")
	reload.Stdout = os.Stdout
	reload.Stderr = os.Stderr
	if err := reload.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: systemctl daemon-reload failed: %v\n", err)
		return
	}

	enable := exec.Command("sudo", "systemctl", "enable", "--now", "huginn")
	enable.Stdout = os.Stdout
	enable.Stderr = os.Stderr
	if err := enable.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: systemctl enable failed: %v\n", err)
		return
	}

	fmt.Println("huginn.service installed, enabled, and started.")
}
