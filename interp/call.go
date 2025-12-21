package interp

import (
	"fmt"
	"slices"

	"github.com/rs/zerolog/log"
	"github.com/timewinder-dev/timewinder/vm"
)

type overlayMain struct {
	*vm.Program
	Main *vm.Function
}

func (o *overlayMain) GetInstruction(ptr vm.ExecPtr) (vm.Op, error) {
	if ptr.CodeID() != 0 {
		return o.Program.GetInstruction(ptr)
	}
	if len(o.Main.Bytecode) <= ptr.Offset() {
		return vm.Op{}, vm.ErrEndOfCode
	}
	return o.Main.Bytecode[ptr.Offset()], nil
}

func FunctionCallFromString(prog *vm.Program, globals *StackFrame, callString string) (*StackFrame, error) {
	callprog, err := vm.CompileExpr(callString)
	if err != nil {
		return nil, err
	}
	overlay := &overlayMain{
		Main:    callprog.Main,
		Program: prog,
	}
	frame := &StackFrame{
		Stack: []vm.Value{},
	}
	for {
		v, n, err := Step(overlay, globals, []*StackFrame{frame})
		if err != nil {
			return nil, err
		}
		if v == CallStep {
			return BuildCallFrame(prog, frame, n)
		}
		if v == ContinueStep {
			continue
		}
		return nil, fmt.Errorf("Calling expression `%s` does not end in a call", callString)
	}
}

func BuildCallFrame(prog *vm.Program, frame *StackFrame, n int) (*StackFrame, error) {
	if len(frame.Stack) < n+1 {
		log.Trace().Int("argc", n).Int("stack_len", len(frame.Stack)).Msg("BuildCallFrame: stack too short")
		return nil, fmt.Errorf("Call stack is too short to buildCallFrame: need %d items, have %d", n+1, len(frame.Stack))
	}

	// Check if calling a builtin function
	fnVal := frame.Pop()
	if builtinVal, ok := fnVal.(vm.BuiltinValue); ok {
		// Look up builtin implementation from registry
		impl, exists := vm.BuiltinRegistry[builtinVal.Name]
		if !exists {
			log.Trace().Str("builtin", builtinVal.Name).Msg("BuildCallFrame: unknown builtin")
			return nil, fmt.Errorf("unknown builtin function: %s", builtinVal.Name)
		}

		// Pop arguments
		args := make([]vm.Value, n)
		for i := n - 1; i >= 0; i-- {
			argVal, ok := frame.Pop().(vm.ArgValue)
			if !ok {
				return nil, fmt.Errorf("Compiler error: stack contains non-call arguments")
			}
			args[i] = argVal.Value
		}

		log.Trace().Str("builtin", builtinVal.Name).Interface("args", args).Msg("BuildCallFrame: calling builtin")

		// Call the builtin implementation
		result, err := impl(args)
		if err != nil {
			log.Trace().Str("builtin", builtinVal.Name).Err(err).Msg("BuildCallFrame: builtin error")
			return nil, err
		}

		log.Trace().Str("builtin", builtinVal.Name).Interface("result", result).Msg("BuildCallFrame: builtin returned")

		// Always push result to stack and increment PC
		// NonDetValue will be handled by RunToPause if in execution context
		// or by Canonicalize if in initialization context
		frame.Push(result)
		frame.PC = frame.PC.Inc()
		return nil, nil // No new call frame for builtins
	}

	// Regular function call - must be FnPtrValue
	fnPtr, ok := fnVal.(vm.FnPtrValue)
	if !ok {
		log.Trace().Interface("fn_val", fnVal).Msg("BuildCallFrame: non-callable value")
		return nil, fmt.Errorf("Compiler error: stack contains non-callable value on call (got %T: %v)", fnVal, fnVal)
	}
	ptr := vm.ExecPtr(fnPtr)
	args := make([]vm.ArgValue, n)
	for i := n - 1; i >= 0; i-- {
		args[i], ok = frame.Pop().(vm.ArgValue)
		if !ok {
			return nil, fmt.Errorf("Compiler error: stack contains non-call arguments")
		}
	}

	fn := prog.Code[ptr.CodeID()-1]
	log.Trace().Str("pc", ptr.String()).Int("argc", n).Interface("args", args).Msg("BuildCallFrame: calling function")

	newFrame := &StackFrame{
		PC:    ptr,
		Stack: []vm.Value{},
	}
	for _, p := range fn.Params {
		found := false
		for i, a := range args {
			if a.Key == p.Name {
				newFrame.StoreVar(p.Name, a.Value)
				args = slices.Delete(args, i, i+1)
				found = true
				break
			}
		}
		if found {
			continue
		}
		if len(args) != 0 {
			a := args[0]
			args = args[1:]
			newFrame.StoreVar(p.Name, a.Value)
			continue
		}
		if p.Default != nil {
			newFrame.StoreVar(p.Name, p.Default)
		} else {
			log.Trace().Str("pc", ptr.String()).Str("param", p.Name).Msg("BuildCallFrame: missing required argument")
			return nil, fmt.Errorf("Not enough arguments to call")
		}
	}

	log.Trace().Str("pc", ptr.String()).Interface("variables", newFrame.Variables).Msg("BuildCallFrame: created call frame")
	return newFrame, nil
}

// BuildMethodCallFrame handles method calls (e.g., arr.append(x))
// Methods execute immediately and don't create new stack frames
func BuildMethodCallFrame(frame *StackFrame, n int) error {
	// Stack layout: arg1, arg2, ..., argN, receiver, methodName
	// Pop method name and receiver
	methodName := mustString(frame.Pop())
	receiver := frame.Pop()

	// Pop arguments (in reverse order)
	args := make([]vm.Value, n)
	for i := n - 1; i >= 0; i-- {
		argVal, ok := frame.Pop().(vm.ArgValue)
		if !ok {
			return fmt.Errorf("Compiler error: stack contains non-call arguments for method call")
		}
		args[i] = argVal.Value
	}

	log.Trace().
		Str("method", methodName).
		Interface("receiver_type", vm.GetTypeName(receiver)).
		Interface("args", args).
		Msg("BuildMethodCallFrame: calling method")

	// Look up method by receiver type
	typeName := vm.GetTypeName(receiver)
	methodTable, ok := vm.MethodRegistry[typeName]
	if !ok {
		return fmt.Errorf("type %s has no methods", typeName)
	}

	method, ok := methodTable[methodName]
	if !ok {
		return fmt.Errorf("type %s has no method %s", typeName, methodName)
	}

	// Call the method
	result, err := method(receiver, args)
	if err != nil {
		log.Trace().Str("method", methodName).Err(err).Msg("BuildMethodCallFrame: method error")
		return err
	}

	log.Trace().Str("method", methodName).Interface("result", result).Msg("BuildMethodCallFrame: method returned")

	// Push result to stack
	frame.Push(result)

	// Increment PC to move past CALL_METHOD instruction
	frame.PC = frame.PC.Inc()

	return nil // No new frame for methods
}
