package core

import (
	"sync"
	"time"
)

type HostState string

const (
	HostStateStarting HostState = "starting"
	HostStateReady     HostState = "ready"
	HostStateUnhealthy HostState = "unhealthy"
	HostStateDraining  HostState = "draining"
)

type HostInfo struct {
	ID            string
	IsPrimary     bool
	State         HostState
	CPUPercent    float64
	MemoryPercent float64
	InstanceCount int
}

type RemovalEvent struct {
	HostID string `json:"hostID"`
	Reason string `json:"reason"`
	Time   int64  `json:"t"`
}

type HostRegistry struct {
	mu             sync.RWMutex
	hosts          map[string]*HostInfo
	recentRemovals []RemovalEvent
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

// Remove deletes the host with no reason recorded.
func (r *HostRegistry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hosts, id)
}

// RemoveWithReason deletes the host and records why, so a connected
// dashboard can show a toast explaining the disappearance.
func (r *HostRegistry) RemoveWithReason(id, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hosts, id)
	r.recentRemovals = append(r.recentRemovals, RemovalEvent{
		HostID: id,
		Reason: reason,
		Time:   time.Now().Unix(),
	})
	if len(r.recentRemovals) > 20 {
		r.recentRemovals = r.recentRemovals[len(r.recentRemovals)-20:]
	}
}

// RecentRemovals returns removals recorded in the last `window`.
func (r *HostRegistry) RecentRemovals(window time.Duration) []RemovalEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cutoff := time.Now().Add(-window).Unix()
	var out []RemovalEvent
	for _, ev := range r.recentRemovals {
		if ev.Time >= cutoff {
			out = append(out, ev)
		}
	}
	return out
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
