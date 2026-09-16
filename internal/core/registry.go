package core

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

type InstanceState string

const (
	StateStarting  InstanceState = "starting"
	StateReady     InstanceState = "ready"
	StateUnhealthy InstanceState = "unhealthy"
	StateDraining  InstanceState = "draining"
)

type Instance struct {
	ID            string
	ContainerID   string
	State         InstanceState
	PlayerCount   int
	MaxPlayers    int
	LastHeartbeat time.Time
	JoinCode      string
	Address       string
	PlayerHistory []int
}

func generateJoinCode() string {
	b := make([]byte, 2)
	rand.Read(b)
	return hex.EncodeToString(b)
}
func (r *Registry) GetByCode(code string) (Instance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, inst := range r.instances {
		if inst.JoinCode == code {
			return *inst, true
		}
	}
	return Instance{}, false
}

type Registry struct {
	mu        sync.RWMutex
	instances map[string]*Instance
	timeout   time.Duration
}

func NewRegistry(heartbeatTimeout time.Duration) *Registry {
	return &Registry{
		instances: make(map[string]*Instance),
		timeout:   heartbeatTimeout,
	}
}
func (r *Registry) SetHeartbeatTimeout(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.timeout = d
}

func (r *Registry) Register(id, containerID, address string, maxPlayers int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances[id] = &Instance{
		ID:            id,
		ContainerID:   containerID,
		Address:       address,
		MaxPlayers:    maxPlayers,
		JoinCode:      generateJoinCode(),
		State:         StateStarting,
		LastHeartbeat: time.Now(),
	}
}
func (r *Registry) Heartbeat(id string, playerCount int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inst, ok := r.instances[id]
	if !ok {
		inst = &Instance{ID: id}
		r.instances[id] = inst
	}
	inst.PlayerCount = playerCount
	inst.LastHeartbeat = time.Now()
	if inst.State != StateDraining {
		inst.State = StateReady
	}
	inst.PlayerHistory = append(inst.PlayerHistory, playerCount)
	const maxHistory = 30
	if len(inst.PlayerHistory) > maxHistory {
		inst.PlayerHistory = inst.PlayerHistory[len(inst.PlayerHistory)-maxHistory:]
	}

}
func (r *Registry) SetState(id string, state InstanceState) {
	inst, ok := r.instances[id]
	if !ok {
		return
	}
	inst.State = state
}
func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.instances, id)
}
func (r *Registry) All() []Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Instance, 0, len(r.instances))
	for _, inst := range r.instances {
		out = append(out, *inst)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (r *Registry) Get(id string) (Instance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	inst, ok := r.instances[id]
	if !ok {
		return Instance{}, false
	}
	return *inst, true
}
func (r *Registry) SweepUnhealthy() []Instance {
	r.mu.Lock()
	defer r.mu.Unlock()
	var newlyUnhealthy []Instance
	cutoff := time.Now().Add(-r.timeout)
	for _, inst := range r.instances {
		if inst.State != StateUnhealthy && inst.LastHeartbeat.Before(cutoff) {
			inst.State = StateUnhealthy
			newlyUnhealthy = append(newlyUnhealthy, *inst)
		}
	}
	return newlyUnhealthy
}
