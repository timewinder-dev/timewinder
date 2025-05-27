package cas

import (
	"bytes"
	"io"

	"github.com/dgryski/go-farm"
)

type MemoryCAS struct {
	data map[Hash]any
}

func NewMemoryCAS() *MemoryCAS {
	return &MemoryCAS{
		data: make(map[Hash]any),
	}
}

func (m *MemoryCAS) getValue(h Hash) (bool, any, error) {
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

func (m *MemoryCAS) Put(item Hashable) (Hash, error) {
	var buf bytes.Buffer
	err := item.Serialize(&buf)
	if err != nil {
		return 0, err
	}
	h := Hash(farm.Hash64(buf.Bytes()))
	m.data[h] = item
	return h, nil
}
