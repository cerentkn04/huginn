package sidecar

import (
	"bufio"
	"context"
	"huginn/internal/types"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"
)

var playerCountRe = regexp.MustCompile(`player_count=(\d+)`)

func RunLogParse(ctx context.Context, cfg Config, sender *UDPSender) error {
	var playerCount int64

	go tailFile(ctx, cfg.LogPath, &playerCount)
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			hb := types.Heartbeat{
				InstanceID:  cfg.InstanceID,
				PlayerCount: int(atomic.LoadInt64(&playerCount)),
				Status:      "ready",
			}
			if err := sender.Send(hb); err != nil {
				log.Printf("sidecar: failed to send heartbeat: %v", err)
				continue
			}
			log.Printf("sidecar: heartbeat sent: instance=%s player_count=%d", hb.InstanceID, hb.PlayerCount)
		}
	}
}
func tailFile(ctx context.Context, path string, playerCount *int64) {
	f, err := waitForFile(ctx, path)
	if err != nil {
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}

		if m := playerCountRe.FindStringSubmatch(line); m != nil {
			if count, err := strconv.Atoi(m[1]); err == nil {
				atomic.StoreInt64(playerCount, int64(count))
			}
		}
	}
}
func waitForFile(ctx context.Context, path string) (*os.File, error) {
	for {
		f, err := os.Open(path)
		if err == nil {
			return f, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}
