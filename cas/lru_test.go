package cas

import (
	"testing"

	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

func TestLRUCache_BasicOperation(t *testing.T) {
	underlying := NewMemoryCAS()
	cache := NewLRUCache(underlying, 3) // Small cache for testing

	// Create test states
	state1 := &interp.State{Globals: &interp.StackFrame{Variables: map[string]vm.Value{"x": vm.IntValue(1)}}}
	state2 := &interp.State{Globals: &interp.StackFrame{Variables: map[string]vm.Value{"x": vm.IntValue(2)}}}
	state3 := &interp.State{Globals: &interp.StackFrame{Variables: map[string]vm.Value{"x": vm.IntValue(3)}}}
	state4 := &interp.State{Globals: &interp.StackFrame{Variables: map[string]vm.Value{"x": vm.IntValue(4)}}}

	// Put states
	hash1, err := cache.Put(state1)
	if err != nil {
		t.Fatalf("Failed to put state1: %v", err)
	}
	hash2, err := cache.Put(state2)
	if err != nil {
		t.Fatalf("Failed to put state2: %v", err)
	}
	hash3, err := cache.Put(state3)
	if err != nil {
		t.Fatalf("Failed to put state3: %v", err)
	}

	// Retrieve state1 - should populate cache
	retrieved1, err := Retrieve[*interp.State](cache, hash1)
	if err != nil {
		t.Fatalf("Failed to retrieve state1: %v", err)
	}
	if retrieved1.Globals.Variables["x"] != vm.IntValue(1) {
		t.Errorf("Retrieved state1 has wrong value: got %v, want %v", retrieved1.Globals.Variables["x"], vm.IntValue(1))
	}

	// Cache should have entries now (state, globals, and sub-objects)
	stats := cache.Stats()
	if stats.Size == 0 {
		t.Errorf("Cache should have entries, got %d", stats.Size)
	}

	// Retrieve state2 and state3
	_, err = Retrieve[*interp.State](cache, hash2)
	if err != nil {
		t.Fatalf("Failed to retrieve state2: %v", err)
	}
	_, err = Retrieve[*interp.State](cache, hash3)
	if err != nil {
		t.Fatalf("Failed to retrieve state3: %v", err)
	}

	// Cache should not exceed max size
	stats = cache.Stats()
	if stats.Size > stats.MaxSize {
		t.Errorf("Cache size %d exceeds max size %d", stats.Size, stats.MaxSize)
	}

	// Add state4 and retrieve - cache should handle evictions automatically
	hash4, err := cache.Put(state4)
	if err != nil {
		t.Fatalf("Failed to put state4: %v", err)
	}
	_, err = Retrieve[*interp.State](cache, hash4)
	if err != nil {
		t.Fatalf("Failed to retrieve state4: %v", err)
	}

	// Cache should not exceed max size even after more retrievals
	stats = cache.Stats()
	if stats.Size > stats.MaxSize {
		t.Errorf("Cache size %d exceeds max size %d after eviction", stats.Size, stats.MaxSize)
	}
}

func TestLRUCache_Has(t *testing.T) {
	underlying := NewMemoryCAS()
	cache := NewLRUCache(underlying, 10)

	state := &interp.State{Globals: &interp.StackFrame{Variables: map[string]vm.Value{"x": vm.IntValue(42)}}}
	hash, err := cache.Put(state)
	if err != nil {
		t.Fatalf("Failed to put state: %v", err)
	}

	if !cache.Has(hash) {
		t.Errorf("Cache should report hash exists")
	}

	if cache.Has(Hash(99999)) {
		t.Errorf("Cache should report non-existent hash doesn't exist")
	}
}
