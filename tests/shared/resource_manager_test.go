//go:build fast
// +build fast

package shared

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestResourceManager_BasicOperations(t *testing.T) {
	t.Parallel()
	rm := NewResourceManager()
	
	rm.RegisterResource("test-resource", 5*time.Second)
	
	ctx := context.Background()
	err := rm.AcquireResource(ctx, "test-resource", "test-owner")
	if err != nil {
		t.Fatalf("Failed to acquire resource: %v", err)
	}
	
	status, err := rm.GetResourceStatus("test-resource")
	if err != nil {
		t.Fatalf("Failed to get resource status: %v", err)
	}
	
	if !status.InUse {
		t.Error("Resource should be in use")
	}
	
	if status.Owner != "test-owner" {
		t.Errorf("Expected owner 'test-owner', got '%s'", status.Owner)
	}
	
	err = rm.ReleaseResource("test-resource", "test-owner")
	if err != nil {
		t.Fatalf("Failed to release resource: %v", err)
	}
	
	status, err = rm.GetResourceStatus("test-resource")
	if err != nil {
		t.Fatalf("Failed to get resource status: %v", err)
	}
	
	if status.InUse {
		t.Error("Resource should not be in use")
	}
	
	if status.Owner != "" {
		t.Errorf("Expected empty owner, got '%s'", status.Owner)
	}
}

func TestResourceManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	rm := NewResourceManager()
	
	rm.RegisterResource("concurrent-resource", 10*time.Second)
	
	const numGoroutines = 10
	var wg sync.WaitGroup
	results := make([]error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			owner := fmt.Sprintf("owner-%d", id)
			err := rm.AcquireResource(ctx, "concurrent-resource", owner)
			results[id] = err
			
			if err == nil {
				// Hold the resource briefly
				time.Sleep(10 * time.Millisecond)
				rm.ReleaseResource("concurrent-resource", owner)
			}
		}(i)
	}
	
	wg.Wait()
	
	successCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		}
	}
	
	if successCount == 0 {
		t.Error("At least one goroutine should have successfully acquired the resource")
	}
}

func TestResourceManager_ResourceQueuing(t *testing.T) {
	t.Parallel()
	rm := NewResourceManager()
	
	rm.RegisterResource("queue-resource", 5*time.Second)
	
	ctx1 := context.Background()
	err := rm.AcquireResource(ctx1, "queue-resource", "owner-1")
	if err != nil {
		t.Fatalf("Failed to acquire resource: %v", err)
	}
	
	var wg sync.WaitGroup
	var secondErr error
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		secondErr = rm.AcquireResource(ctx2, "queue-resource", "owner-2")
		if secondErr == nil {
			rm.ReleaseResource("queue-resource", "owner-2")
		}
	}()
	
	time.Sleep(100 * time.Millisecond)
	
	err = rm.ReleaseResource("queue-resource", "owner-1")
	if err != nil {
		t.Fatalf("Failed to release resource: %v", err)
	}
	
	wg.Wait()
	
	if secondErr != nil {
		t.Errorf("Second goroutine should have acquired resource after first released: %v", secondErr)
	}
}

func TestResourceManager_ContextCancellation(t *testing.T) {
	t.Parallel()
	rm := NewResourceManager()
	
	rm.RegisterResource("cancel-resource", 10*time.Second)
	
	ctx1 := context.Background()
	err := rm.AcquireResource(ctx1, "cancel-resource", "owner-1")
	if err != nil {
		t.Fatalf("Failed to acquire resource: %v", err)
	}
	
	ctx2, cancel := context.WithCancel(context.Background())
	
	var wg sync.WaitGroup
	var secondErr error
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		secondErr = rm.AcquireResource(ctx2, "cancel-resource", "owner-2")
	}()
	
	time.Sleep(100 * time.Millisecond)
	cancel()
	
	wg.Wait()
	
	if secondErr != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", secondErr)
	}
	
	rm.ReleaseResource("cancel-resource", "owner-1")
}

func TestResourceManager_ForceRelease(t *testing.T) {
	t.Parallel()
	rm := NewResourceManager()
	
	rm.RegisterResource("force-resource", 5*time.Second)
	
	ctx := context.Background()
	err := rm.AcquireResource(ctx, "force-resource", "owner-1")
	if err != nil {
		t.Fatalf("Failed to acquire resource: %v", err)
	}
	
	err = rm.ForceReleaseResource("force-resource")
	if err != nil {
		t.Fatalf("Failed to force release resource: %v", err)
	}
	
	status, err := rm.GetResourceStatus("force-resource")
	if err != nil {
		t.Fatalf("Failed to get resource status: %v", err)
	}
	
	if status.InUse {
		t.Error("Resource should not be in use after force release")
	}
	
	if status.Owner != "" {
		t.Errorf("Expected empty owner after force release, got '%s'", status.Owner)
	}
}

func TestResourceManager_ListResources(t *testing.T) {
	t.Parallel()
	rm := NewResourceManager()
	
	rm.RegisterResource("resource-1", 5*time.Second)
	rm.RegisterResource("resource-2", 10*time.Second)
	
	ctx := context.Background()
	err := rm.AcquireResource(ctx, "resource-1", "owner-1")
	if err != nil {
		t.Fatalf("Failed to acquire resource: %v", err)
	}
	
	resources := rm.ListResources()
	
	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
	
	if resource1, exists := resources["resource-1"]; exists {
		if !resource1.InUse {
			t.Error("Resource-1 should be in use")
		}
		if resource1.Owner != "owner-1" {
			t.Errorf("Expected owner 'owner-1', got '%s'", resource1.Owner)
		}
	} else {
		t.Error("Resource-1 should exist in list")
	}
	
	if resource2, exists := resources["resource-2"]; exists {
		if resource2.InUse {
			t.Error("Resource-2 should not be in use")
		}
		if resource2.Owner != "" {
			t.Errorf("Expected empty owner, got '%s'", resource2.Owner)
		}
	} else {
		t.Error("Resource-2 should exist in list")
	}
	
	rm.ReleaseResource("resource-1", "owner-1")
}