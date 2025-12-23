package cas

import (
	"bytes"
	"io"
	"sort"
	"sync"

	"github.com/dgryski/go-farm"
	"github.com/timewinder-dev/timewinder/interp"
)

type MemoryCAS struct {
	mu              sync.RWMutex
	data            map[Hash][]byte
	weakStateDepths map[Hash][]int // Track depths where each weak state hash was seen
}

func NewMemoryCAS() *MemoryCAS {
	return &MemoryCAS{
		data:            make(map[Hash][]byte),
		weakStateDepths: make(map[Hash][]int),
	}
}

func (m *MemoryCAS) getValue(h Hash) (bool, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[h]
	if !ok {
		return false, nil, nil
	}
	return true, v, nil
}

func (m *MemoryCAS) getReader(h Hash) (bool, io.Reader, error) {
	return false, nil, nil
}

func (m *MemoryCAS) Hash(hash Hash) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[hash]
	return ok
}

func (m *MemoryCAS) Has(hash Hash) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[hash]
	return ok
}

func (m *MemoryCAS) Put(item Hashable) (Hash, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Special handling for State: decompose into nested hash references
	if state, ok := item.(*interp.State); ok {
		return decomposeState(m, state)
	}

	// For other types, store directly as before
	var buf bytes.Buffer
	err := item.Serialize(&buf)
	if err != nil {
		return 0, err
	}
	data := buf.Bytes()
	h := Hash(farm.Hash64(data))
	m.data[h] = data
	return h, nil
}

// RecordWeakStateDepth records that a weak state hash was seen at the given depth
func (m *MemoryCAS) RecordWeakStateDepth(weakHash Hash, depth int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.weakStateDepths[weakHash] = append(m.weakStateDepths[weakHash], depth)
	sort.Ints(m.weakStateDepths[weakHash])
}

// GetWeakStateDepths returns all depths where the given weak state hash was seen
func (m *MemoryCAS) GetWeakStateDepths(weakHash Hash) []int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to avoid race conditions
	depths := m.weakStateDepths[weakHash]
	result := make([]int, len(depths))
	copy(result, depths)
	return result
}
