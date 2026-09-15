package core

import (
	"context"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func RunReclaimLoop(ctx context.Context, cli *client.Client, reg *Registry, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reclaimUnhealthy(ctx, cli, reg)
		}
	}
}

func reclaimUnhealthy(ctx context.Context, cli *client.Client, reg *Registry) {
	for _, inst := range reg.All() {
		if inst.State != "unhealthy" {
			continue
		}

		log.Printf("huginn: reclaiming unhealthy instance %s (container %s)", inst.ID, inst.ContainerID)

		if err := cli.ContainerStop(ctx, inst.ContainerID, container.StopOptions{}); err != nil {
			log.Printf("huginn: reclaim: stop failed for %s: %v", inst.ID, err)
		}
		if err := cli.ContainerRemove(ctx, inst.ContainerID, types.ContainerRemoveOptions{Force: true}); err != nil {
			log.Printf("huginn: reclaim: remove failed for %s: %v", inst.ID, err)
			continue
		}

		reg.Remove(inst.ID)
	}
}
