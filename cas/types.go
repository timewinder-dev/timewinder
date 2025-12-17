package cas

import (
	"fmt"
	"io"
	"reflect"

	"github.com/shamaton/msgpack/v2"
	"github.com/timewinder-dev/timewinder/interp"
)

// TypedEntry wraps a Hashable with a type tag for deserialization
type TypedEntry struct {
	TypeTag string
	Data    []byte
}

func (t *TypedEntry) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, t)
}

func (t *TypedEntry) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, t)
}

// typeRegistry maps type tags to reflect.Type for deserialization
var typeRegistry = make(map[string]reflect.Type)

// Register a type in the registry
func registerType(tag string, example Hashable) {
	typeRegistry[tag] = reflect.TypeOf(example)
}

func init() {
	// Register all reference types (internal CAS formats)
	// These are the only types that need to be registered, as they implement Hashable
	registerType("StateRef", &StateRef{})
	registerType("StackFrameRef", &StackFrameRef{})
	registerType("IteratorStateRef", &IteratorStateRef{})
	registerType("SliceIteratorData", &SliceIteratorData{})
	registerType("DictIteratorData", &DictIteratorData{})
	registerType("StructValueRef", &StructValueRef{})
	registerType("ArrayValueRef", &ArrayValueRef{})

	// Register State type (implements Hashable)
	registerType("State", &interp.State{})

	// Note: vm.Value types and other interp types are NOT registered here
	// They don't implement the full Hashable interface (missing Deserialize)
	// They are handled specially by putValueDirect/getValueTypeTag
}

// getTypeTag returns the type tag for a given item
func getTypeTag(item Hashable) string {
	t := reflect.TypeOf(item)

	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check against registered types
	for tag, regType := range typeRegistry {
		checkType := regType
		if checkType.Kind() == reflect.Ptr {
			checkType = checkType.Elem()
		}
		if t == checkType {
			return tag
		}
	}

	// Fallback: use type name
	return t.Name()
}

// createInstance creates a new instance of the registered type
func createInstance(tag string) (Hashable, error) {
	regType, ok := typeRegistry[tag]
	if !ok {
		return nil, fmt.Errorf("unknown type tag: %s", tag)
	}

	// Create new instance
	// If regType is a pointer type, create instance of pointee
	if regType.Kind() == reflect.Ptr {
		elem := regType.Elem()
		instance := reflect.New(elem).Interface()
		return instance.(Hashable), nil
	}

	// If regType is a value type, create pointer to new instance
	instance := reflect.New(regType).Elem().Interface()

	// Try to convert to Hashable
	if h, ok := instance.(Hashable); ok {
		return h, nil
	}

	// If direct conversion failed, try pointer
	ptrInstance := reflect.New(regType).Interface()
	if h, ok := ptrInstance.(Hashable); ok {
		return h, nil
	}

	return nil, fmt.Errorf("type %s does not implement Hashable", tag)
}
