package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"crypto/rand"
	"encoding/hex"
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
cloud_provider: gcp
auth_token : %s
`, game, image, minInstances, maxInstances, maxPlayers, port,token)

	if err := os.WriteFile("config.yaml", []byte(cfg), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "huginn: failed to write config.yaml: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Wrote config.yaml — run `huginn start config.yaml` to launch your fleet.")
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

