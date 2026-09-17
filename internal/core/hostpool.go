package core

import (
	"fmt"
	"sync"

	"github.com/docker/docker/client"
)

type HostPool struct {
	mu    sync.RWMutex
	hosts map[string]*client.Client
}

func NewHostPool() *HostPool {
	return &HostPool{hosts: make(map[string]*client.Client)}
}

func (p *HostPool) Add(hostID string, cli *client.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hosts[hostID] = cli
}

func (p *HostPool) Get(hostID string) (*client.Client, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cli, ok := p.hosts[hostID]
	if !ok {
		return nil, fmt.Errorf("hostpool: no client for host %q", hostID)
	}
	return cli, nil
}

func (p *HostPool) Remove(hostID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.hosts, hostID)
}

func (p *HostPool) HostIDs() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]string, 0, len(p.hosts))
	for id := range p.hosts {
		ids = append(ids, id)
	}
	return ids
}
