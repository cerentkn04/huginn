package sidecar
import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"
	"encoding/json"
	"huginn/internal/types"
)

func RunSDKHeartbeats(ctx context.Context, cfg Config, sender *UDPSender, playerCount *int64) error {
	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			hb := types.Heartbeat{
				InstanceID:  cfg.InstanceID,
				PlayerCount: int(atomic.LoadInt64(playerCount)),
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
func NewHTTPServer(cfg Config, sender *UDPSender, playerCount *int64) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CurrentPlayers int `json:"current_players"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		atomic.StoreInt64(playerCount, int64(req.CurrentPlayers))
		w.WriteHeader(http.StatusOK)
	})

	return &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}
}
