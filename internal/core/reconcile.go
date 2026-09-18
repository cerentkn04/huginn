package core

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)
type DiscoveredHost struct {
	HostID     string
	Containers []types.Container
}
func InstanceFromContainer(c types.Container, cfg Config, publicHost string) (id string, containerID string, address string, ok bool) {
	instanceID, exists := c.Labels["huginn.instance_id"]
	if !exists {
		return "", "", "", false
	}

	var hostPort int
	for _, p := range c.Ports {
		if p.Type == "udp" && int(p.PrivatePort) == cfg.GamePort {
			hostPort = int(p.PublicPort)
			break
		}
	}
	if hostPort == 0 {
		return "", "", "", false
	}

	address  = fmt.Sprintf("%s:%d", publicHost, hostPort)
	return instanceID, c.ID, address, true
}
func DiscoverInstances(ctx context.Context, cli *client.Client) ([]types.Container, error) {
	f := filters.NewArgs()
	f.Add("label", "huginn.managed=true")

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, fmt.Errorf("docker: listing managed containers: %w", err)
	}
	return containers, nil
}

func DiscoverAllHosts(ctx context.Context, hostPool *HostPool) ([]DiscoveredHost, error) {
	var results []DiscoveredHost
	for _, hostID := range hostPool.HostIDs() {
		cli, err := hostPool.Get(hostID)
		if err != nil {
			continue
		}
		containers, err := DiscoverInstances(ctx, cli)
		if err != nil {
			return nil, fmt.Errorf("discovering on host %s: %w", hostID, err)
		}
		results = append(results, DiscoveredHost{HostID: hostID, Containers: containers})
	}
	return results, nil
}

