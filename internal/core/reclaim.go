package core

import (
	"context"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func StopAndRemove(ctx context.Context, cli *client.Client, containerID string) error {
	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		log.Printf("huginn: stop failed for %s: %v", containerID[:12], err)
	}
	return cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
}

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

		log.Printf("huginn: reclaiming unhealthy instance %s (container %s)", inst.ID, inst.ContainerID[:12])
		if err := StopAndRemove(ctx, cli, inst.ContainerID); err != nil {
			log.Printf("huginn: reclaim: remove failed for %s: %v", inst.ID, err)
			continue
		}
		reg.Remove(inst.ID)
	}
}

func ShutdownAll(cli *client.Client, reg *Registry, timeout time.Duration) {
	instances := reg.All()
	if len(instances) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("huginn: stopping %d instance(s)", len(instances))
	for _, inst := range instances {
		if err := StopAndRemove(ctx, cli, inst.ContainerID); err != nil {
			log.Printf("huginn: shutdown: failed to remove %s: %v", inst.ID, err)
			continue
		}
		reg.Remove(inst.ID)
	}
	log.Println("huginn: all instances stopped")
}
