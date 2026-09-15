package core

import (
	"context"
	"github.com/docker/docker/client"
	"log"
	"time"
)

func RunScaleDown(ctx context.Context, cli *client.Client, store *ConfigStore, reg *Registry) error {
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
					if err := StopInstance(ctx, cli, draining.ContainerID); err != nil {
						log.Printf("huginn: scale-down: failed to stop %s: %v", draining.ID, err)
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
