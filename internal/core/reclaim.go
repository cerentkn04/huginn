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
		log.Printf("huginn: stop failed for %s: %v", shortID(containerID), err)
	}
	return cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{Force: true})
}
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
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
