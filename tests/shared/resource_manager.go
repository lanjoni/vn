package shared

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ResourceManager struct {
	mu        sync.Mutex
	resources map[string]*Resource
	waiters   map[string][]chan struct{}
}

type Resource struct {
	Name      string
	InUse     bool
	Owner     string
	AcquiredAt time.Time
	MaxWait   time.Duration
}

func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		resources: make(map[string]*Resource),
		waiters:   make(map[string][]chan struct{}),
	}
}

func (rm *ResourceManager) RegisterResource(name string, maxWait time.Duration) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.resources[name] = &Resource{
		Name:    name,
		InUse:   false,
		MaxWait: maxWait,
	}
}

func (rm *ResourceManager) AcquireResource(ctx context.Context, name, owner string) error {
	rm.mu.Lock()
	
	resource, exists := rm.resources[name]
	if !exists {
		rm.mu.Unlock()
		return fmt.Errorf("resource %s not registered", name)
	}
	
	if !resource.InUse {
		resource.InUse = true
		resource.Owner = owner
		resource.AcquiredAt = time.Now()
		rm.mu.Unlock()
		return nil
	}
	
	// Resource is in use, need to wait
	waiter := make(chan struct{})
	rm.waiters[name] = append(rm.waiters[name], waiter)
	rm.mu.Unlock()
	
	// Wait for resource to become available or context to be cancelled
	select {
	case <-waiter:
		// Try to acquire again
		return rm.AcquireResource(ctx, name, owner)
	case <-ctx.Done():
		// Remove ourselves from waiters
		rm.removeWaiter(name, waiter)
		return ctx.Err()
	case <-time.After(resource.MaxWait):
		rm.removeWaiter(name, waiter)
		return fmt.Errorf("timeout waiting for resource %s", name)
	}
}

func (rm *ResourceManager) ReleaseResource(name, owner string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	resource, exists := rm.resources[name]
	if !exists {
		return fmt.Errorf("resource %s not registered", name)
	}
	
	if !resource.InUse {
		return fmt.Errorf("resource %s is not in use", name)
	}
	
	if resource.Owner != owner {
		return fmt.Errorf("resource %s is owned by %s, not %s", name, resource.Owner, owner)
	}
	
	resource.InUse = false
	resource.Owner = ""
	resource.AcquiredAt = time.Time{}
	
	// Notify next waiter
	if waiters := rm.waiters[name]; len(waiters) > 0 {
		close(waiters[0])
		rm.waiters[name] = waiters[1:]
	}
	
	return nil
}

func (rm *ResourceManager) removeWaiter(name string, waiter chan struct{}) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	waiters := rm.waiters[name]
	for i, w := range waiters {
		if w == waiter {
			rm.waiters[name] = append(waiters[:i], waiters[i+1:]...)
			break
		}
	}
}

func (rm *ResourceManager) GetResourceStatus(name string) (*Resource, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	resource, exists := rm.resources[name]
	if !exists {
		return nil, fmt.Errorf("resource %s not registered", name)
	}
	
	// Return a copy to avoid race conditions
	return &Resource{
		Name:       resource.Name,
		InUse:      resource.InUse,
		Owner:      resource.Owner,
		AcquiredAt: resource.AcquiredAt,
		MaxWait:    resource.MaxWait,
	}, nil
}

func (rm *ResourceManager) ListResources() map[string]*Resource {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	result := make(map[string]*Resource)
	for name, resource := range rm.resources {
		result[name] = &Resource{
			Name:       resource.Name,
			InUse:      resource.InUse,
			Owner:      resource.Owner,
			AcquiredAt: resource.AcquiredAt,
			MaxWait:    resource.MaxWait,
		}
	}
	
	return result
}

func (rm *ResourceManager) ForceReleaseResource(name string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	resource, exists := rm.resources[name]
	if !exists {
		return fmt.Errorf("resource %s not registered", name)
	}
	
	resource.InUse = false
	resource.Owner = ""
	resource.AcquiredAt = time.Time{}
	
	// Notify next waiter
	if waiters := rm.waiters[name]; len(waiters) > 0 {
		close(waiters[0])
		rm.waiters[name] = waiters[1:]
	}
	
	return nil
}

func (rm *ResourceManager) Cleanup() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	// Close all waiting channels
	for name, waiters := range rm.waiters {
		for _, waiter := range waiters {
			close(waiter)
		}
		rm.waiters[name] = nil
	}
	
	// Reset all resources
	for _, resource := range rm.resources {
		resource.InUse = false
		resource.Owner = ""
		resource.AcquiredAt = time.Time{}
	}
}