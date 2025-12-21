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
