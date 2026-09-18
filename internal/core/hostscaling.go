package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type HostUtilization struct {
	HostID           string
	InstanceCount    int
	CPUPercent       float64
	MemoryPercent    float64
	MemoryUsedBytes  uint64
	MemoryTotalBytes int64
}

func hostNeedsRelief(u HostUtilization, threshold float64) bool {
	return u.CPUPercent > threshold || u.MemoryPercent > threshold
}

func waitForSSH(hostID, zone string) error {
	for i := 0; i < 30; i++ {
		time.Sleep(10 * time.Second)
		cmd := exec.Command("gcloud", "compute", "ssh", hostID, "--zone="+zone, "--command=cat /tmp/huginn-provision-done")
		if out, err := cmd.CombinedOutput(); err == nil {
			return nil
		} else if i == 29 {
			return fmt.Errorf("host never became ready: %v\n%s", err, out)
		}
	}
	return fmt.Errorf("unreachable")
}

func RunHostScalingLoop(ctx context.Context, hostPool *HostPool, store *ConfigStore, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var provisioning bool
	var mu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			mu.Lock()
			if provisioning {
				mu.Unlock()
				continue
			}
			mu.Unlock()

			cfg := store.Get()
			results, err := GetPoolUtilization(ctx, hostPool)
			if err != nil {
				log.Printf("huginn: host-scaling: utilization check failed: %v", err)
				continue
			}

			allFull := len(results) > 0
			for _, u := range results {
				if !hostNeedsRelief(u, cfg.HostScaleUpThreshold) {
					allFull = false
					break
				}
			}

			if !allFull {
				continue
			}

			mu.Lock()
			provisioning = true
			mu.Unlock()

			go func() {
				defer func() {
					mu.Lock()
					provisioning = false
					mu.Unlock()
				}()
				log.Printf("huginn: host-scaling: all hosts over threshold, provisioning a new host...")

				newHostID := fmt.Sprintf("huginn-host-%d", time.Now().Unix())
				inst, err := CreateHost(ctx, cfg.GCPProject, cfg.GCPZone, newHostID)
				if err != nil {
					log.Printf("huginn: host-scaling: failed to create host: %v", err)
					return
				}
				ip := InternalIP(inst)

				certsDir := "certs"
				caCertPath := certsDir + "/ca.pem"
				caKeyPath := certsDir + "/ca-key.pem"

				if err := waitForSSH(newHostID, cfg.GCPZone); err != nil {
					log.Printf("huginn: host-scaling: host never became ready: %v", err)
					return
				}

				certPEM, keyPEM, err := GenerateServerCert(caCertPath, caKeyPath, ip)
				if err != nil {
					log.Printf("huginn: host-scaling: cert generation failed: %v", err)
					return
				}
				if err := SetupRemoteTLS(cfg.GCPZone, newHostID, caCertPath, certPEM, keyPEM); err != nil {
					log.Printf("huginn: host-scaling: TLS setup failed: %v", err)
					return
				}

				clientCertPath := certsDir + "/client-cert.pem"
				clientKeyPath := certsDir + "/client-key.pem"
				cli, err := client.NewClientWithOpts(
					client.WithHost(fmt.Sprintf("tcp://%s:2376", ip)),
					client.WithTLSClientConfig(caCertPath, clientCertPath, clientKeyPath),
					client.WithAPIVersionNegotiation(),
				)
				if err != nil {
					log.Printf("huginn: host-scaling: docker client failed: %v", err)
					return
				}

				hostPool.Add(newHostID, cli)
				log.Printf("huginn: host-scaling: new host %s ready and added to pool", newHostID)
			}()
		}
	}
}

func GetPoolUtilization(ctx context.Context, hostPool *HostPool) ([]HostUtilization, error) {
	var results []HostUtilization
	for _, hostID := range hostPool.HostIDs() {
		cli, err := hostPool.Get(hostID)
		if err != nil {
			continue
		}
		util, err := GetHostUtilization(ctx, hostID, cli)
		if err != nil {
			return nil, err
		}
		results = append(results, util)
	}
	return results, nil
}

func GetHostUtilization(ctx context.Context, hostID string, cli *client.Client) (HostUtilization, error) {
	info, err := cli.Info(ctx)
	if err != nil {
		return HostUtilization{}, fmt.Errorf("host %s: getting docker info: %w", hostID, err)
	}

	containers, err := DiscoverInstances(ctx, cli)
	if err != nil {
		return HostUtilization{}, fmt.Errorf("host %s: listing containers: %w", hostID, err)
	}

	util := HostUtilization{
		HostID:           hostID,
		InstanceCount:    len(containers),
		MemoryTotalBytes: info.MemTotal,
	}

	for _, c := range containers {
		cpuPct, memUsed, err := containerStats(ctx, cli, c.ID)
		if err != nil {
			continue
		}
		util.CPUPercent += cpuPct
		util.MemoryUsedBytes += memUsed
	}

	if info.MemTotal > 0 {
		util.MemoryPercent = float64(util.MemoryUsedBytes) / float64(info.MemTotal) * 100
	}

	return util, nil
}

func containerStats(ctx context.Context, cli *client.Client, containerID string) (cpuPercent float64, memUsedBytes uint64, err error) {
	resp, err := cli.ContainerStats(ctx, containerID, true)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var first, second types.StatsJSON
	if err := decoder.Decode(&first); err != nil {
		return 0, 0, err
	}
	if err := decoder.Decode(&second); err != nil {
		return 0, 0, err
	}

	cpuDelta := float64(second.CPUStats.CPUUsage.TotalUsage) - float64(first.CPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(second.CPUStats.SystemUsage) - float64(first.CPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		numCPUs := float64(second.CPUStats.OnlineCPUs)
		if numCPUs == 0 {
			numCPUs = float64(len(second.CPUStats.CPUUsage.PercpuUsage))
		}
		cpuPercent = (cpuDelta / systemDelta) * numCPUs * 100.0
	}

	memUsedBytes = second.MemoryStats.Usage
	return cpuPercent, memUsedBytes, nil
}
