package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"huginn/internal/sidecar"
)

func main() {
	cfg := sidecar.LoadConfig()

	sender, err := sidecar.NewUDPSender(cfg.CoreUDPAddr)
	if err != nil {
		log.Fatalf("sidecar: failed to init UDP sender: %v", err)
	}
	defer sender.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	var playerCount int64

	srv := sidecar.NewHTTPServer(cfg, sender, &playerCount)

	go func() {
		log.Printf("sidecar: instance=%s mode=%s HTTP on %s, heartbeats -> %s",
			cfg.InstanceID, cfg.Mode, cfg.HTTPAddr, cfg.CoreUDPAddr)
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("sidecar: HTTP server stopped: %v", err)
		}
	}()

	switch cfg.Mode {
	case "log_parse":
		log.Printf("sidecar: tailing %s for player_count updates", cfg.LogPath)
		if err := sidecar.RunLogParse(ctx, cfg, sender); err != nil && ctx.Err() == nil {
			log.Fatalf("sidecar: log_parse mode failed: %v", err)
		}
	case "sdk":
		if err := sidecar.RunSDKHeartbeats(ctx, cfg, sender, &playerCount); err != nil && ctx.Err() == nil {
			log.Fatalf("sidecar: sdk mode failed: %v", err)
		}
	default:
		log.Fatalf("sidecar: unknown SIDECAR_MODE %q (want log_parse or sdk)", cfg.Mode)
	}

	log.Println("sidecar: shutting down")
}
