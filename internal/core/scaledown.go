package core

import (
	"context"
	"log"
	"time"
)

func RunScaleDown(ctx context.Context, hostPool *HostPool, store *ConfigStore, reg *Registry) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			cfg := store.Get()
			found := false
			var draining Instance
			for _, inst := range reg.All() {
				if inst.State == StateDraining {
					draining = inst
					found = true
					break
				}
			}
			if found {
				if draining.PlayerCount == 0 {
					cli, err := hostPool.Get(draining.HostID)
					if err != nil {
						log.Printf("huginn: scale-down: host %s for instance %s is gone, dropping stale registry entry", draining.HostID, draining.ID)
						reg.Remove(draining.ID)
						continue
					}
					if err := StopInstance(ctx, cli, draining.ContainerID); err != nil {
						log.Printf("huginn: scale-down: failed to stop %s (treating as already gone): %v", draining.ID, err)
						reg.Remove(draining.ID)
						continue
					}
					reg.Remove(draining.ID)
					log.Printf("huginn: scaled down: removed instance %s", draining.ID)
				} else {
					reg.SetState(draining.ID, StateReady)
					log.Printf("huginn: scale-down: aborted draining %s, a player joined", draining.ID)
				}
				continue
			}
			available := 0
			total := 0
			for _, inst := range reg.All() {
				total++
				if inst.State == StateReady && inst.PlayerCount < cfg.MaxPlayers {
					available++
				}
			}
			if available <= cfg.BufferSize || total <= cfg.MinInstances {
				continue
			}

			for _, inst := range reg.All() {
				if inst.State == StateReady && inst.PlayerCount == 0 {
					reg.SetState(inst.ID, StateDraining)
					log.Printf("huginn: scale-down: marked %s draining", inst.ID)
					break
				}
			}
		}
	}
}
