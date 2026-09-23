package core

import (
        "context"
        "encoding/json"
        "fmt"
        "github.com/goccy/go-yaml"
        "huginn/internal/web"
        "io/fs"
        "log"
        "net"
        "net/http"
        "strconv"
	"strings"
        "os"
	"crypto/subtle"
        "time"
)
type summary struct {
	TotalInstances int
	TotalPlayer    int
}

func NewRestServer(ctx context.Context, hostPool *HostPool,  hostRegistry *HostRegistry,store *ConfigStore, reg *Registry) *http.Server {
	mux := http.NewServeMux()
	ApiAvailable(mux, reg)
	ApiFleet(mux, reg)
	ApiCode(mux, reg)
	ApiStop(mux, ctx, hostPool, reg)
	ApiRestart(mux, ctx, hostPool, store, reg)
	ApiFleetStream(mux, reg)
	ApiServers(mux, reg)
	LogFleetStream(mux, reg,hostPool)
	ApiConfig(mux, store, reg)
	ApiHosts(mux, hostRegistry)
	ApiCreateHost(mux, ctx, hostPool, hostRegistry, store)
	ApiHostsStream(mux, hostRegistry)
	if err := ServeDashboard(mux); err != nil {
		log.Printf("huginn: dashboard unavailable: %v", err)
	}
	return &http.Server{Addr: store.Get().HTTPListenAddr, Handler: requireAuth(store.Get().AuthToken, store.Get().ClientAuthToken, mux)}
}
func ApiAvailable(mux *http.ServeMux, reg *Registry) {
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
func ApiFleet(mux *http.ServeMux, reg *Registry) {
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

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

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


func ApiServers(mux *http.ServeMux, reg *Registry) {
	mux.HandleFunc("/api/servers", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(reg.All())
	})

}
func writeHostSnapshot(w http.ResponseWriter, flusher http.Flusher, hostRegistry *HostRegistry) error {
	payload, err := json.Marshal(hostRegistry.All())
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}

	removals := hostRegistry.RecentRemovals(20 * time.Second)
	if len(removals) > 0 {
		removalsPayload, err := json.Marshal(removals)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: removed\ndata: %s\n\n", removalsPayload); err != nil {
			return err
		}
	}

	flusher.Flush()
	return nil
}
func requireAuth(adminToken, clientToken string, next http.Handler) http.Handler {
	if adminToken == "" {
		log.Fatal("huginn: auth_token is not set in config — refusing to start with no authentication. Run `huginn init` or add auth_token manually.")
	}
	clientOnlyPaths := []string{
		"/api/servers/available",
		"/api/servers/code/",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		provided := r.Header.Get("Authorization")
		provided = strings.TrimPrefix(provided, "Bearer ")
		if provided == "" {
			provided = r.URL.Query().Get("token")
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(adminToken)) == 1 {
			next.ServeHTTP(w, r)
			return
		}

		if clientToken != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(clientToken)) == 1 {
			for _, p := range clientOnlyPaths {
				if strings.HasPrefix(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden: client token cannot access this endpoint", http.StatusForbidden)
			return
		}

		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}
func ApiCreateHost(mux *http.ServeMux, ctx context.Context, hostPool *HostPool, hostRegistry *HostRegistry, store *ConfigStore) {
	mux.HandleFunc("/api/hosts/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cfg := store.Get()
		if cfg.GCPProject == "" || cfg.GCPZone == "" {
			http.Error(w, "GCP project/zone not configured", http.StatusBadRequest)
			return
		}
		newHostID := fmt.Sprintf("huginn-host-%d", time.Now().Unix())
		go func() {
			if err := ProvisionHost(ctx, hostPool, hostRegistry, cfg, newHostID); err != nil {
				log.Printf("huginn: manual host create: %v", err)
			}
		}()
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"hostID": newHostID, "status": "provisioning"})
	})
}
func ApiStop(mux *http.ServeMux, ctx context.Context, hostPool *HostPool, reg *Registry) {
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
		cli, err := hostPool.Get(inst.HostID)
		if err != nil {
			log.Printf("huginn: api: stop: %v", err)
			http.Error(w, "host unavailable", http.StatusInternalServerError)
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
func ApiRestart(mux *http.ServeMux, ctx context.Context, hostPool *HostPool, store *ConfigStore, reg *Registry) {
	mux.HandleFunc("/api/servers/restart/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Path[len("/api/servers/restart/"):]
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		inst, ok := reg.Get(id)
		if !ok {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		_, portStr, err := net.SplitHostPort(inst.Address)
		if err != nil {
			log.Printf("huginn: api: restart: bad address %q for %s: %v", inst.Address, inst.ID, err)
			http.Error(w, "failed to restart server", http.StatusInternalServerError)
			return
		}
		hostPort, err := strconv.Atoi(portStr)
		if err != nil {
			log.Printf("huginn: api: restart: bad port %q for %s: %v", portStr, inst.ID, err)
			http.Error(w, "failed to restart server", http.StatusInternalServerError)
			return
		}
		cli, err := hostPool.Get(inst.HostID)
		if err != nil {
			log.Printf("huginn: api: restart: %v", err)
			http.Error(w, "host unavailable", http.StatusInternalServerError)
			return
		}
		if err := StopInstance(ctx, cli, inst.ContainerID); err != nil {
			log.Printf("huginn: api: restart: failed to stop old container for %s: %v", inst.ID, err)
			http.Error(w, "failed to restart server", http.StatusInternalServerError)
			return
		}
		reg.Remove(inst.ID)

		cfg := store.Get()
		containerID, err := StartInstance(ctx, cli, cfg, inst.ID, hostPort)
		if err != nil {
			log.Printf("huginn: api: restart: failed to start new container for %s: %v", inst.ID, err)
			http.Error(w, "failed to restart server", http.StatusInternalServerError)
			return
		}
		reg.Register(inst.ID, containerID,inst.HostID,inst.Address, inst.MaxPlayers)

		newInst, _ := reg.Get(inst.ID)
		log.Printf("huginn: api: restarted instance %s via REST", inst.ID)
		json.NewEncoder(w).Encode(newInst)
	})
}
func ApiCode(mux *http.ServeMux, reg *Registry) {
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
func ServeDashboard(mux *http.ServeMux) error {
	dist, err := fs.Sub(web.Files, "dist")
	if err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(dist)))
	return nil
}
func ApiHosts(mux *http.ServeMux, hostRegistry *HostRegistry) {
	mux.HandleFunc("/api/hosts", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(hostRegistry.All())
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
	if _, err := fmt.Fprintf(w, "event: peak\ndata: %d\n\n", reg.SampleFleetPeak()); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
func ApiHostsStream(mux *http.ServeMux, hostRegistry *HostRegistry) {
	mux.HandleFunc("/api/hosts/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		if err := writeHostSnapshot(w, flusher, hostRegistry); err != nil {
			log.Printf("huginn: hosts sse: initial write failed: %v", err)
			return
		}

		for {
			select {
			case <-r.Context().Done():
				log.Printf("huginn: hosts sse: client disconnected")
				return
			case <-ticker.C:
				if err := writeHostSnapshot(w, flusher, hostRegistry); err != nil {
					log.Printf("huginn: hosts sse: write failed: %v", err)
					return
				}
			}
		}
	})
}

func ApiConfig(mux *http.ServeMux, store *ConfigStore, reg *Registry) {
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(store.Get())

		case http.MethodPost:
			newCfg := store.Get()
			if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
				log.Printf("huginn: api: bad config payload: %v", err)
				http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			if err := newCfg.validate(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := writeConfig(newCfg.path, newCfg); err != nil {
				log.Printf("huginn: api: failed to write config: %v", err)
				http.Error(w, "failed to save config", http.StatusInternalServerError)
				return
			}
			store.Set(newCfg)
			reg.SetHeartbeatTimeout(time.Duration(newCfg.HeartbeatTimeout) * time.Second)
			go EnsureFirewall(context.Background(), newCfg)
			log.Printf("huginn: api: config saved to %s", newCfg.path)
			json.NewEncoder(w).Encode(newCfg)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func writeConfig(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}
