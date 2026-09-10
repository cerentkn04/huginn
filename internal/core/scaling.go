package core

import (
	"context"
	"fmt"
	"log"
	"time"
	"github.com/docker/docker/client"
)

func RunScalingLoop(ctx context.Context, cli *client.Client, cfg Config, reg *Registry) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	nextIndex := cfg.MinInstances

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			available := 0
			total := 0
			for _, inst := range reg.All() {
				total++
				if inst.State == StateReady && inst.PlayerCount < cfg.MaxPlayers {
					available++
				}
			}

			if available >= cfg.BufferSize || total >= cfg.MaxInstances {
				continue
			}

			instanceID := fmt.Sprintf("huginn-inst-%d", nextIndex)
			hostPort := cfg.GamePort + nextIndex
		

			containerID, err := StartInstance(ctx, cli, cfg, instanceID, hostPort)
			if err != nil {
				log.Printf("huginn: scale-up: failed to start %s: %v", instanceID, err)
				continue
			}
			nextIndex++
			address := fmt.Sprintf("%s:%d", cfg.PublicHost, hostPort)
			reg.Register(instanceID, containerID, address, cfg.MaxPlayers)
			log.Printf("huginn: scaled up: started instance %s on host port %d (container %s)", instanceID, hostPort, containerID[:12])
		}
	}
}
