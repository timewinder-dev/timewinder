package vm

import "fmt"

// MethodImpl represents a method implementation
// It takes the receiver value and arguments, returns the new receiver value (for mutation) and any error
type MethodImpl func(receiver Value, args []Value) (Value, error)

// MethodTable maps method names to their implementations for a specific type
type MethodTable map[string]MethodImpl

// MethodRegistry maps type names to their method tables
var MethodRegistry = map[string]MethodTable{
	"array": {
		"append": arrayAppend,
		"pop":    arrayPop,
	},
}

// GetTypeName returns the type name for a value (for method dispatch)
func GetTypeName(v Value) string {
	switch v.(type) {
	case ArrayValue:
		return "array"
	case StructValue:
		return "struct"
	case IntValue:
		return "int"
	case FloatValue:
		return "float"
	case StrValue:
		return "string"
	case BoolValue:
		return "bool"
	case NoneValue:
		return "none"
	default:
		return "unknown"
	}
}

// arrayAppend implements the .append() method for arrays
// This performs IN-PLACE mutation by returning a new array with the element appended
func arrayAppend(receiver Value, args []Value) (Value, error) {
	arr, ok := receiver.(ArrayValue)
	if !ok {
		return nil, fmt.Errorf("append called on non-array: %T", receiver)
	}

	if len(args) != 1 {
		return nil, fmt.Errorf("append expects 1 argument, got %d", len(args))
	}

	// Create new array with appended element (Go's append does this efficiently)
	newArr := append(arr, args[0])
	return newArr, nil
}

// arrayPop implements the .pop() method for arrays
// pop() - removes and returns last element
// pop(index) - removes and returns element at index
// Returns a tuple (new_array, popped_value) but in our system we return the new array
// and the popped value needs to be handled separately
func arrayPop(receiver Value, args []Value) (Value, error) {
	arr, ok := receiver.(ArrayValue)
	if !ok {
		return nil, fmt.Errorf("pop called on non-array: %T", receiver)
	}

	if len(arr) == 0 {
		return nil, fmt.Errorf("pop from empty array")
	}

	var index int
	if len(args) == 0 {
		// No argument - pop last element
		index = len(arr) - 1
	} else if len(args) == 1 {
		// Pop at specific index
		idxVal, ok := args[0].(IntValue)
		if !ok {
			return nil, fmt.Errorf("pop index must be integer, got %T", args[0])
		}
		index = int(idxVal)

		// Handle negative indices Python-style
		if index < 0 {
			index = len(arr) + index
		}

		// Bounds check
		if index < 0 || index >= len(arr) {
			return nil, fmt.Errorf("pop index %d out of bounds for array of length %d", int(idxVal), len(arr))
		}
	} else {
		return nil, fmt.Errorf("pop expects 0 or 1 argument, got %d", len(args))
	}

	// Create new array without the element at index
	newArr := make(ArrayValue, 0, len(arr)-1)
	newArr = append(newArr, arr[:index]...)
	newArr = append(newArr, arr[index+1:]...)

	return newArr, nil
}
