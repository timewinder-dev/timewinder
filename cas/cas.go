package cas

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/timewinder-dev/timewinder/interp"
)

type CAS interface {
	Put(item Hashable) (Hash, error)
	Has(hash Hash) bool
	getReader(hash Hash) (bool, io.Reader, error)

	// Weak state tracking for livelock detection
	RecordWeakStateDepth(weakHash Hash, depth int)
	GetWeakStateDepths(weakHash Hash) []int
}

type Serde interface {
	Serialize(w io.Writer) error
	Deserialize(r io.Reader) error
}

type Hashable interface {
	Serde
}

type directStore interface {
	getValue(h Hash) (bool, []byte, error)
}

type Hash uint64

func Retrieve[T Hashable](c CAS, hash Hash) (T, error) {
	var t T
	v, ok := c.(directStore)
	if !ok {
		return t, errors.New("CAS does not support direct retrieval")
	}

	has, data, err := v.getValue(hash)
	if err != nil {
		return t, err
	}
	if !has {
		return t, fmt.Errorf("hash not found in CAS: %d", hash)
	}

	// Check if we're retrieving a State - handle recomposition
	var zeroT T
	targetType := reflect.TypeOf(zeroT)
	if targetType == reflect.TypeOf((*interp.State)(nil)) {
		// Recompose State from StateRef
		state, err := recomposeState(v, hash)
		if err != nil {
			return t, fmt.Errorf("recomposing State: %w", err)
		}
		return any(state).(T), nil
	}

	// For other types, deserialize directly from bytes
	// First deserialize the TypedEntry to get the actual data
	typedEntry := &TypedEntry{}
	buf := bytes.NewReader(data)
	err = typedEntry.Deserialize(buf)
	if err != nil {
		return t, fmt.Errorf("deserializing TypedEntry: %w", err)
	}

	// Create instance and deserialize
	instance, err := createInstance(typedEntry.TypeTag)
	if err != nil {
		return t, fmt.Errorf("creating instance: %w", err)
	}

	dataBuf := bytes.NewReader(typedEntry.Data)
	err = instance.Deserialize(dataBuf)
	if err != nil {
		return t, fmt.Errorf("deserializing data: %w", err)
	}

	result, ok := instance.(T)
	if !ok {
		return t, fmt.Errorf("type mismatch: expected %T, got %T", t, instance)
	}

	return result, nil
}
