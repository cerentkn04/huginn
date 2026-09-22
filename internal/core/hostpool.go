package core

import (
	"fmt"
	"sync"

	"github.com/docker/docker/client"
)

type HostPool struct {
	mu    sync.RWMutex
	hosts map[string]*client.Client
	publicIPs map[string]string
	order   []string
	nextIdx int
}

func NewHostPool() *HostPool {
	return &HostPool{
		hosts: make(map[string]*client.Client),
		publicIPs: make(map[string]string),
	}
}

func (p *HostPool) Add(hostID string, cli *client.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.hosts[hostID]; !exists {
		p.order = append(p.order, hostID)
	}
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
func (p *HostPool) SelectHost() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.order) == 0 {
		return "", fmt.Errorf("hostpool: no hosts available")
	}
	id := p.order[p.nextIdx%len(p.order)]
	p.nextIdx++
	return id, nil
}
func (p *HostPool) Remove(hostID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.hosts, hostID)
	for i, id := range p.order {
		if id == hostID {
			p.order = append(p.order[:i], p.order[i+1:]...)
			break
		}
	}
}
func (p *HostPool) Has(hostID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.hosts[hostID]
	return ok
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

func (p *HostPool) SetPublicIP(hostID, ip string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.publicIPs[hostID] = ip
}

func (p *HostPool) GetPublicIP(hostID string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ip, ok := p.publicIPs[hostID]
	if !ok || ip == "" {
		return "", fmt.Errorf("hostpool: no public IP known for host %q", hostID)
	}
	return ip, nil
}
