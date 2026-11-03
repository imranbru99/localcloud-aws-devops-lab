package services

import (
	"fmt"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	adapters map[string]ServiceAdapter
}

var DefaultRegistry = &Registry{
	adapters: make(map[string]ServiceAdapter),
}

func Register(adapter ServiceAdapter) {
	DefaultRegistry.mu.Lock()
	defer DefaultRegistry.mu.Unlock()
	DefaultRegistry.adapters[adapter.Name()] = adapter
}

func Get(name string) (ServiceAdapter, bool) {
	DefaultRegistry.mu.RLock()
	defer DefaultRegistry.mu.RUnlock()
	a, ok := DefaultRegistry.adapters[name]
	return a, ok
}

func All() map[string]ServiceAdapter {
	DefaultRegistry.mu.RLock()
	defer DefaultRegistry.mu.RUnlock()
	out := make(map[string]ServiceAdapter)
	for k, v := range DefaultRegistry.adapters {
		out[k] = v
	}
	return out
}

func ResetAll(account, region string) error {
	DefaultRegistry.mu.Lock()
	defer DefaultRegistry.mu.Unlock()
	for name, a := range DefaultRegistry.adapters {
		if err := a.Reset(account, region); err != nil {
			return fmt.Errorf("error resetting service %s: %w", name, err)
		}
	}
	return nil
}
