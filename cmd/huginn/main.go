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
	if len(os.Args) < 3 || os.Args[1] != "start" {
		fmt.Fprintln(os.Stderr, "usage: huginn start <config.yaml>")
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

	log.Printf("huginn: pulling image %s", cfg.Image)
	if err := core.PullImage(ctx, cli, cfg.Image); err != nil {
		log.Fatalf("huginn: %v", err)
	}
	cfg.PublicHost = core.DiscoverPublicHost(cfg)
	if cfg.PublicHost == "" {
		log.Println("huginn: WARNING: ...")
	}

	store := core.NewConfigStore(cfg)
discovered, err := core.DiscoverInstances(ctx, cli)
if err != nil {
	log.Fatalf("huginn: failed to discover existing containers: %v", err)
}

adopted := make(map[string]bool)
for _, c := range discovered {
	id, containerID, address, ok := core.InstanceFromContainer(c, cfg, cfg.PublicHost)
	if !ok {
		continue
	}
	reg.Register(id, containerID, address, cfg.MaxPlayers)
	adopted[id] = true
	log.Printf("huginn: adopted existing instance %s (container %s)", id, containerID[:12])
}

for i := 0; i < cfg.MinInstances; i++ {
	instanceID := fmt.Sprintf("huginn-inst-%d", i)
	if adopted[instanceID] {
		continue
	}
	hostPort := cfg.GamePort + i
	containerID, err := core.StartInstance(ctx, cli, cfg, instanceID, hostPort)
	if err != nil {
		log.Printf("huginn: instance %s failed to start: %v — rolling back", instanceID, err)
		for _, inst := range reg.All() {
			if stopErr := core.StopInstance(ctx, cli, inst.ContainerID); stopErr != nil {
				log.Printf("huginn: rollback: failed to stop %s: %v", inst.ID, stopErr)
			}
		}
		os.Exit(1)
	}
	address := fmt.Sprintf("%s:%d", cfg.PublicHost, hostPort)
	reg.Register(instanceID, containerID, address, cfg.MaxPlayers)
	log.Printf("huginn: started instance %s on host port %d (container %s)", instanceID, hostPort, containerID[:12])
}
		
	log.Printf("huginn: %d instance(s) running, heartbeats on %s", cfg.MinInstances, cfg.UDPListenAddr)
	go func() {
		srv := core.NewRestServer(ctx, cli, store, reg)
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("huginn: REST API server failed: %v", err)
		}
	}()
	go core.EnsureFirewall(ctx, cfg)
	go func() {
		if err := core.RunScalingLoop(ctx, cli, store, reg); err != nil && ctx.Err() == nil {
			log.Printf("huginn: scaling loop failed: %v", err)
		}
	}()
	go func() {
		if err := core.RunScaleDown(ctx, cli, store, reg); err != nil && ctx.Err() == nil {
			log.Printf("huginn: scale-down loop failed: %v", err)
		}
	}()
	go core.RunReclaimLoop(ctx, cli, reg, 10*time.Second)
	<-ctx.Done()

	log.Println("huginn: shutting down")
	core.ShutdownAll(cli, reg, 30*time.Second)
}
