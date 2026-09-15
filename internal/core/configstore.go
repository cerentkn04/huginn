package core

import "sync"

// ConfigStore is a thread-safe holder for the current Config, so a config
// change saved through the dashboard applies to already-running loops
// immediately instead of requiring a restart.
type ConfigStore struct {
	mu  sync.RWMutex
	cfg Config
}

func NewConfigStore(cfg Config) *ConfigStore {
	return &ConfigStore{cfg: cfg}
}

func (s *ConfigStore) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *ConfigStore) Set(cfg Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}
