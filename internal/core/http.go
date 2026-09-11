package core

import (
	"encoding/json"
	"net/http"
	"context"
	"log"
	"fmt"
	"time"
	"github.com/docker/docker/client"
)

type summary struct {
	TotalInstances int
	TotalPlayer    int
}

func NewRestServer(ctx context.Context , cli *client.Client, cfg Config, reg *Registry) *http.Server {
	mux := http.NewServeMux()
	ApiAvailable(mux,reg)
	ApiFleet(mux,reg)
	ApiCode(mux,reg)
	ApiStop(mux,ctx,cli,reg)
	ApiFleetStream(mux, reg)
	ApiServers(mux,reg)
	return &http.Server{Addr: cfg.HTTPListenAddr, Handler: mux}
}
func ApiAvailable(mux *http.ServeMux,reg *Registry){
	mux.HandleFunc("/api/servers/available", func(w http.ResponseWriter, r *http.Request) {
		for _, inst := range reg.All() {
			if inst.State == StateReady && inst.PlayerCount < inst.MaxPlayers {
				json.NewEncoder(w).Encode(inst)
				return
			}
		}
		http.Error(w, "no available servers", http.StatusServiceUnavailable)
	})

}
func ApiFleet(mux *http.ServeMux, reg *Registry){
		mux.HandleFunc("/api/fleet", func(w http.ResponseWriter, r *http.Request) {
		players := 0
		instances := 0
		for _, inst := range reg.All() {
			instances++
			players += inst.PlayerCount
		}
		json.NewEncoder(w).Encode(summary{TotalInstances: instances, TotalPlayer: players})
	})


}
func ApiFleetStream(mux *http.ServeMux, reg *Registry) {
	mux.HandleFunc("/api/fleet/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// send one immediately so the dashboard isn't blank for the first second
		if err := writeSnapshot(w, flusher, reg); err != nil {
			log.Printf("huginn: sse: initial write failed: %v", err)
			return
		}

		for {
			select {
			case <-r.Context().Done():
				log.Printf("huginn: sse: client disconnected")
				return
			case <-ticker.C:
				if err := writeSnapshot(w, flusher, reg); err != nil {
					log.Printf("huginn: sse: write failed: %v", err)
					return
				}
			}
		}
	})
}

func writeSnapshot(w http.ResponseWriter, flusher http.Flusher, reg *Registry) error {
	payload, err := json.Marshal(reg.All())
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
func ApiServers(mux *http.ServeMux, reg *Registry){
	mux.HandleFunc("/api/servers", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(reg.All())
	})

}
func ApiStop(mux *http.ServeMux, ctx context.Context, cli *client.Client, reg *Registry) {
	mux.HandleFunc("/api/servers/stop/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
    			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
  			 return
		}
		id := r.URL.Path[len("/api/servers/stop/"):]
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		inst, ok := reg.Get(id)
		if !ok {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		if err := StopInstance(ctx, cli, inst.ContainerID); err != nil {
			log.Printf("huginn: api: failed to stop instance %s: %v", inst.ID, err)
			http.Error(w, "failed to stop server", http.StatusInternalServerError)
			return
		}
		reg.Remove(inst.ID)
		log.Printf("huginn: api: stopped instance %s via REST", inst.ID)
		json.NewEncoder(w).Encode(inst)
	})    

    }

func ApiCode(mux *http.ServeMux , reg *Registry){
	mux.HandleFunc("/api/servers/code/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Path[len("/api/servers/code/"):]
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		inst, ok := reg.GetByCode(code)
		if !ok {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(inst)
	})


}
