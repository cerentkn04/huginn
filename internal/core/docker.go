package core

import (
	"context"
	"fmt"
	"io"
	"net"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)
func NewDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker: creating client: %w", err)
	}
	return cli, nil
}

func PullImage(ctx context.Context, cli *client.Client, imageName string) error {
	if _, _, err := cli.ImageInspectWithRaw(ctx, imageName); err == nil {
		return nil // already present locally
	}

	reader, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("docker: pulling %s: %w", imageName, err)
	}
	defer reader.Close()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("docker: reading pull progress for %s: %w", imageName, err)
	}
	return nil
}
func StartInstance(ctx context.Context, cli *client.Client, cfg Config, instanceID string, hostPort int) (containerID string, err error) {
	containerPort, err := nat.NewPort("udp", fmt.Sprintf("%d", cfg.GamePort))
	if err != nil {
		return "", fmt.Errorf("docker: invalid container port %d: %w", cfg.GamePort, err)
	}

	_, corePort, err := net.SplitHostPort(cfg.UDPListenAddr)
	if err != nil {
		return "", fmt.Errorf("docker: invalid udp_listen_addr %q: %w", cfg.UDPListenAddr, err)
	}

	containerCfg := &container.Config{
		Image: cfg.Image,
		Env: []string{
			"SIDECAR_INSTANCE_ID=" + instanceID,
			"SIDECAR_CORE_UDP_ADDR=host.docker.internal:" + corePort,
		"SIDECAR_MODE=" + cfg.Mode,
		
		},
		ExposedPorts: nat.PortSet{containerPort: struct{}{}},
	}

	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", hostPort)}},
		},
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
	}

	created, err := cli.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, instanceID)
	if err != nil {
		return "", fmt.Errorf("docker: creating container %s: %w", instanceID, err)
	}
if err := cli.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}); err != nil {
    _ = cli.ContainerRemove(ctx, created.ID, types.ContainerRemoveOptions{Force: true})
    return "", fmt.Errorf("docker: starting container %s: %w", instanceID, err)
}

	return created.ID, nil
}

func StopInstance(ctx context.Context, cli *client.Client, containerID string) error {
	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return fmt.Errorf("docker: stopping container %s: %w", containerID, err)
	}
	if err := cli.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{}); err != nil {
		return fmt.Errorf("docker: removing container %s: %w", containerID, err)
	}
	return nil
}
