package cas

import (
	"bytes"
	"io"

	"github.com/dgryski/go-farm"
	"github.com/timewinder-dev/timewinder/interp"
)

type MemoryCAS struct {
	data map[Hash][]byte
}

func NewMemoryCAS() *MemoryCAS {
	return &MemoryCAS{
		data: make(map[Hash][]byte),
	}
}

func (m *MemoryCAS) getValue(h Hash) (bool, []byte, error) {
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
	_, ok := m.data[hash]
	return ok
}

func (m *MemoryCAS) Has(hash Hash) bool {
	_, ok := m.data[hash]
	return ok
}

func (m *MemoryCAS) Put(item Hashable) (Hash, error) {
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
