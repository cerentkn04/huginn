package core

import "sync"

type HostState string

const (
	HostStateStarting   HostState = "starting"
	HostStateReady      HostState = "ready"
	HostStateUnhealthy  HostState = "unhealthy"
	HostStateDraining   HostState = "draining" // reserved — no scale-down logic sets this yet
)

type HostInfo struct {
	ID            string
	State         HostState
	CPUPercent    float64
	MemoryPercent float64
	InstanceCount int
}

type HostRegistry struct {
	mu    sync.RWMutex
	hosts map[string]*HostInfo
}

func NewHostRegistry() *HostRegistry {
	return &HostRegistry{hosts: make(map[string]*HostInfo)}
}

func (r *HostRegistry) Update(info HostInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hosts[info.ID] = &info
}

func (r *HostRegistry) SetState(id string, state HostState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.hosts[id]; ok {
		h.State = state
		return
	}
	r.hosts[id] = &HostInfo{ID: id, State: state}
}

func (r *HostRegistry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hosts, id)
}

func (r *HostRegistry) All() []HostInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]HostInfo, 0, len(r.hosts))
	for _, h := range r.hosts {
		out = append(out, *h)
	}
	return out
}
