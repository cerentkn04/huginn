package main

import (
	"context"
	"fmt"
	"huginn/internal/core"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: huginn <init|start> [args]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		runInit()
		return
	case "start":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: huginn start <config.yaml>")
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: huginn <init|start> [args]")
		os.Exit(1)
	}
	cfg, err := core.LoadConfig(os.Args[2])
	if err != nil {
		log.Fatalf("huginn: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	reg := core.NewRegistry(time.Duration(cfg.HeartbeatTimeout) * time.Second)
	go func() {
		if err := core.RunUDPListener(cfg.UDPListenAddr, reg); err != nil && ctx.Err() == nil {
			log.Fatalf("huginn: UDP listener failed: %v", err)
		}
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, inst := range reg.SweepUnhealthy() {
					log.Printf("huginn: instance %s marked unhealthy (no heartbeat for %ds)", inst.ID, cfg.HeartbeatTimeout)
				}
			}
		}
	}()
	cli, err := core.NewDockerClient()
	if err != nil {
		log.Fatalf("huginn: %v", err)
	}

	defer cli.Close()
	hostPool := core.NewHostPool()
	hostPool.Add("gamegin", cli)
	go func() {
		hosts, err := core.ListManagedHosts(ctx, cfg.GCPProject, cfg.GCPZone)
		if err != nil {
			log.Printf("huginn: could not check for orphaned hosts: %v", err)
			return
		}
		for _, h := range hosts {
			name := h.GetName()
			if name == "" || hostPool.Has(name) {
				continue
			}
			log.Printf("huginn: WARNING — found orphaned GCP host %q (not known to this huginn instance) — it may be costing money with nothing managing it. Delete manually if unwanted: gcloud compute instances delete %s --zone=%s", name, name, cfg.GCPZone)
		}
	}()
	hostRegistry := core.NewHostRegistry()

	log.Printf("huginn: pulling image %s", cfg.Image)
	if err := core.PullImage(ctx, cli, cfg.Image); err != nil {
		log.Fatalf("huginn: %v", err)
	}
	cfg.InternalHost = core.DiscoverInternalHost(cfg)
		if cfg.InternalHost == "" {
		log.Println("huginn: WARNING: could not determine internal host address; sidecar heartbeats may fail")
	}
	cfg.PublicHost = core.DiscoverPublicHost(cfg)
	if cfg.PublicHost == "" {
		log.Println("huginn: WARNING: ...")
	}

	store := core.NewConfigStore(cfg)
discoveredHosts, err := core.DiscoverAllHosts(ctx, hostPool)
if err != nil {
	log.Fatalf("huginn: failed to discover existing containers: %v", err)
}

adopted := make(map[string]bool)
for _, dh := range discoveredHosts {
	for _, c := range dh.Containers {
		id, containerID, address, ok := core.InstanceFromContainer(c, cfg, cfg.PublicHost)
		if !ok {
			continue
		}
		reg.Register(id, containerID, dh.HostID, address, cfg.MaxPlayers)
		adopted[id] = true
		log.Printf("huginn: adopted existing instance %s on host %s (container %s)", id, dh.HostID, containerID[:12])
	}
}

for i := 0; i < cfg.MinInstances; i++ {
	instanceID := fmt.Sprintf("huginn-inst-%d", i)
	if adopted[instanceID] {
		continue
	}
	hostPort := cfg.GamePort + i
	hostID, err := hostPool.SelectHost()
		if err != nil {
		log.Fatalf("huginn: %v", err)
	}
	hostCli, err := hostPool.Get(hostID)
		if err != nil {
		log.Fatalf("huginn: %v", err)
	}
	containerID, err := core.StartInstance(ctx, hostCli, cfg, instanceID, hostPort)
	if err != nil {
		log.Printf("huginn: instance %s failed to start: %v — rolling back", instanceID, err)
		for _, inst := range reg.All() {
			if stopErr := core.StopInstance(ctx, hostCli, inst.ContainerID); stopErr != nil {
				log.Printf("huginn: rollback: failed to stop %s: %v", inst.ID, stopErr)
			}
		}
		os.Exit(1)
	}
	address := fmt.Sprintf("%s:%d", cfg.PublicHost, hostPort)
	reg.Register(instanceID, containerID,hostID, address, cfg.MaxPlayers)
	log.Printf("huginn: started instance %s on host port %d (container %s)", instanceID, hostPort, containerID[:12])
}
		
	log.Printf("huginn: %d instance(s) running, heartbeats on %s", cfg.MinInstances, cfg.UDPListenAddr)
	go func() {
		srv := core.NewRestServer(ctx, hostPool, hostRegistry, store, reg)
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("huginn: REST API server failed: %v", err)
		}
	}()
	go core.EnsureFirewall(ctx, cfg)
	go func() {
		if err := core.RunScalingLoop(ctx, hostPool, store, reg); err != nil && ctx.Err() == nil {
			log.Printf("huginn: scaling loop failed: %v", err)
		}
	}()
	go func() {
		if err := core.RunScaleDown(ctx, cli, store, reg); err != nil && ctx.Err() == nil {
			log.Printf("huginn: scale-down loop failed: %v", err)
		}
	}()
		go func() {
		if err := core.RunHostScalingLoop(ctx, hostPool, hostRegistry, store, 15*time.Second); err != nil && ctx.Err() == nil {
			log.Printf("huginn: host-scaling loop failed: %v", err)
		}
	}()
	go core.RunReclaimLoop(ctx, cli, reg, 10*time.Second)
	<-ctx.Done()

	log.Println("huginn: shutting down")
	core.ShutdownAll(cli, reg, 30*time.Second)
}
