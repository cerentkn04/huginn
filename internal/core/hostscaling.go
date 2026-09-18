package core

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type HostUtilization struct {
	HostID            string
	InstanceCount     int
	CPUPercent        float64
	MemoryPercent     float64
	MemoryUsedBytes   uint64
	MemoryTotalBytes  int64
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
			continue // a container mid-stop/mid-start shouldn't kill the whole check
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
