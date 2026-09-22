package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
	"os"
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
func waitForSSH(ctx context.Context, projectID, hostID, zone string) error {
	for i := 0; i < 30; i++ {
		time.Sleep(10 * time.Second)
		cmd := exec.Command("gcloud", "compute", "ssh", hostID, "--zone="+zone, "--ssh-key-file="+os.Getenv("HOME")+"/.ssh/huginn_automation_key", "--command=cat /tmp/huginn-provision-done")
		if out, err := cmd.CombinedOutput(); err == nil {
			return nil
		} else if i == 29 {
			return fmt.Errorf("host never became ready: %v\n%s", err, out)
		}

		if i%3 == 2 {
			exists, existsErr := HostExists(ctx, projectID, zone, hostID)
			if existsErr == nil && !exists {
				return fmt.Errorf("host %s no longer exists in GCP (deleted externally)", hostID)
			}
		}
	}
	return fmt.Errorf("unreachable")
}
func ProvisionHost(ctx context.Context, hostPool *HostPool, hostRegistry *HostRegistry, cfg Config, newHostID string) error {
	log.Printf("huginn: provisioning new host %s...", newHostID)
	hostRegistry.SetState(newHostID, HostStateStarting)

	inst, err := CreateHost(ctx, cfg.GCPProject, cfg.GCPZone, newHostID)
	if err != nil {
		hostRegistry.Remove(newHostID)
		return fmt.Errorf("failed to create host: %w", err)
	}
	ip := InternalIP(inst)
	externalIP := ExternalIP(inst)
	hostPool.SetPublicIP(newHostID, externalIP)
	certsDir := "certs"
	caCertPath := certsDir + "/ca.pem"
	caKeyPath := certsDir + "/ca-key.pem"

	if err := waitForSSH(ctx, cfg.GCPProject, newHostID, cfg.GCPZone); err != nil {
		hostRegistry.Remove(newHostID)
		return fmt.Errorf("host never became ready: %w", err)
	}

	certPEM, keyPEM, err := GenerateServerCert(caCertPath, caKeyPath, ip)
	if err != nil {
		hostRegistry.Remove(newHostID)
		return fmt.Errorf("cert generation failed: %w", err)
	}
	if err := SetupRemoteTLS(cfg.GCPZone, newHostID, caCertPath, certPEM, keyPEM); err != nil {
		hostRegistry.Remove(newHostID)
		return fmt.Errorf("TLS setup failed: %w", err)
	}

	clientCertPath := certsDir + "/client-cert.pem"
	clientKeyPath := certsDir + "/client-key.pem"
	cli, err := client.NewClientWithOpts(
		client.WithHost(fmt.Sprintf("tcp://%s:2376", ip)),
		client.WithTLSClientConfig(caCertPath, clientCertPath, clientKeyPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		hostRegistry.Remove(newHostID)
		return fmt.Errorf("docker client failed: %w", err)
	}
	if err := PullImage(ctx, cli, cfg.Image);err != nil{
	 hostRegistry.Remove(newHostID)
	 return fmt.Errorf("image pull failed: %w",err)
	}

	hostPool.Add(newHostID, cli)
	hostRegistry.SetState(newHostID, HostStateReady)
	log.Printf("huginn: host %s ready and added to pool", newHostID)
	return nil
}
func RunHostScalingLoop(ctx context.Context, hostPool *HostPool, hostRegistry *HostRegistry,store *ConfigStore, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var busy bool
	var mu sync.Mutex
	idleSince := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			mu.Lock()
			if busy {
				mu.Unlock()
				continue
			}
			mu.Unlock()
			starting := 0
			for _, h := range hostRegistry.All() {
				if h.State == HostStateStarting {starting++}
			}
			if starting > 0 {continue}
		
			cfg := store.Get()
			if !cfg.HostAutoScalingEnabled {
				continue
			}
                        results, failed := GetPoolUtilization(ctx, hostPool)
                        for hostID, ferr := range failed {
                                log.Printf("huginn: host-scaling: utilization check failed for %s: %v", hostID, ferr)
                                if hostPool.IsPrimary(hostID) || hostPool.IsDraining(hostID) {
                                        continue
                                }
                                exists, existsErr := HostExists(ctx, cfg.GCPProject, cfg.GCPZone, hostID)
                                if existsErr == nil && !exists {
                                        log.Printf("huginn: host-scaling: host %s no longer exists in GCP, cleaning up stale pool entry", hostID)
                                        if cli, cerr := hostPool.Get(hostID); cerr == nil && cli != nil {
                                                cli.Close()
                                        }
                                        hostPool.Remove(hostID)
                                        hostRegistry.Remove(hostID)
                                        delete(idleSince, hostID)
                                }
                        }
			for _, u := range results {
				hostRegistry.Update(HostInfo{
					ID:            u.HostID,
					State:         HostStateReady,
					CPUPercent:    u.CPUPercent,
					MemoryPercent: u.MemoryPercent,
					InstanceCount: u.InstanceCount,
				})
			}

			allFull := len(results) > 0
			for _, u := range results {
				if !hostNeedsRelief(u, cfg.HostScaleUpThreshold) {
					allFull = false
					break
				}
			}
			if allFull {
				mu.Lock()
				busy = true
				mu.Unlock()
				go func() {
					defer func() {
						mu.Lock()
						busy = false
						mu.Unlock()
					}()
					log.Printf("huginn: host-scaling: all hosts over threshold, provisioning a new host...")
					newHostID := fmt.Sprintf("huginn-host-%d", time.Now().Unix())
					if err := ProvisionHost(ctx, hostPool, hostRegistry, cfg, newHostID); err != nil {
						log.Printf("huginn: host-scaling: %v", err)
					}
				}()
				continue
			}
			idleThreshold := time.Duration(cfg.HostScaleDownIdleMinutes) * time.Minute
			if idleThreshold <= 0 {
				idleThreshold = 10 * time.Minute
			}

			var drainCandidate string
			now := time.Now()
			for _, u := range results {
				if hostPool.IsPrimary(u.HostID) {
					continue
				}
				if u.InstanceCount == 0 {
					since, ok := idleSince[u.HostID]
					if !ok {
						idleSince[u.HostID] = now
						continue
					}
					if now.Sub(since) >= idleThreshold {
						drainCandidate = u.HostID
						break
					}
				} else {
					delete(idleSince, u.HostID)
				}
			}

			if drainCandidate == "" {
				continue
			}
			mu.Lock()
			busy= true
			mu.Unlock()
			go func(hostID string) {
				defer func() {
					mu.Lock()
					busy= false
					mu.Unlock()
				}()
				delete(idleSince, hostID)
				if err := drainAndRemoveHost(ctx, hostPool, hostRegistry, cfg, hostID); err != nil {
					log.Printf("huginn: host-scaling: drain %s failed: %v", hostID, err)
				}
			}(drainCandidate)
						

		}
	}
}

func GetPoolUtilization(ctx context.Context, hostPool *HostPool) ([]HostUtilization,  map[string]error) {
	var results []HostUtilization
	failed := make(map[string]error)
	for _, hostID := range hostPool.HostIDs() {
		cli, err := hostPool.Get(hostID)
		if err != nil {
			 failed[hostID] = err
			continue
		}
		util, err := GetHostUtilization(ctx, hostID, cli)
		if err != nil {
			failed[hostID] = err
			continue
		}
		results = append(results, util)
	}
	return results, failed
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
func drainAndRemoveHost(ctx context.Context, hostPool *HostPool, hostRegistry *HostRegistry, cfg Config, hostID string) error {
	log.Printf("huginn: host-scaling: draining idle host %s...", hostID)
	hostPool.SetDraining(hostID, true)
	hostRegistry.SetState(hostID, HostStateDraining)

	cli, err := hostPool.Get(hostID)
	if err == nil {
		if util, uerr := GetHostUtilization(ctx, hostID, cli); uerr == nil && util.InstanceCount > 0 {
			hostPool.SetDraining(hostID, false)
			hostRegistry.SetState(hostID, HostStateReady)
			return fmt.Errorf("abandoned drain: host %s picked up %d instance(s) since being marked idle", hostID, util.InstanceCount)
		}
	}

	if err := DeleteHost(ctx, cfg.GCPProject, cfg.GCPZone, hostID); err != nil {
		hostPool.SetDraining(hostID, false)
		hostRegistry.SetState(hostID, HostStateReady)
		return fmt.Errorf("failed to delete GCP host: %w", err)
	}

	if cli != nil {
		cli.Close()
	}
	hostPool.Remove(hostID)
	hostRegistry.Remove(hostID)
	log.Printf("huginn: host-scaling: host %s drained and removed", hostID)
	return nil
}
