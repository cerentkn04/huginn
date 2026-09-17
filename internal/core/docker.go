package core

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"strings"
	"net"
)

func NewDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker: creating client: %w", err)
	}
	return cli, nil
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
			"SIDECAR_CORE_UDP_ADDR=" + cfg.InternalHost + ":" + corePort,
			"SIDECAR_MODE=" + cfg.Mode,
		},
		ExposedPorts: nat.PortSet{containerPort: struct{}{}},
		Labels: map[string]string{
        		"huginn.managed":     "true",
        		"huginn.instance_id": instanceID,
   		},
	}

	hostCfg := &container.HostConfig{
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", hostPort)}},
		},
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
func PullImage(ctx context.Context, cli *client.Client, imageName string) error {
	if _, _, err := cli.ImageInspectWithRaw(ctx, imageName); err == nil {
		return nil
	}

	reader, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return friendlyPullError(imageName, err)
	}
	defer reader.Close()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return friendlyPullError(imageName, err)
	}
	return nil
}

func friendlyPullError(imageName string, err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "pull access denied"), strings.Contains(msg, "repository does not exist"):
		return fmt.Errorf(
			"could not find image %q — check the spelling in your config's `image` field, "+
				"or if it's a private image, run `docker login` first (original error: %w)",
			imageName, err,
		)
	case strings.Contains(msg, "manifest unknown"), strings.Contains(msg, "not found"):
		return fmt.Errorf(
			"image %q was found but the tag doesn't exist — check the tag (e.g. \":latest\") is correct (original error: %w)",
			imageName, err,
		)
	case strings.Contains(msg, "Cannot connect to the Docker daemon"):
		return fmt.Errorf(
			"could not reach Docker — is the Docker daemon running? (original error: %w)",
			err,
		)
	default:
		return fmt.Errorf("failed to pull image %q: %w", imageName, err)
	}
}
