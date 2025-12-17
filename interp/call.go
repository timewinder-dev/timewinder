package interp

import (
	"fmt"
	"slices"

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
	frame := &StackFrame{}
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
		return nil, fmt.Errorf("Call stack is too short to buildCallFrame: need %d items, have %d", n+1, len(frame.Stack))
	}

	// Check if calling a builtin function
	fnVal := frame.Pop()
	if builtinVal, ok := fnVal.(vm.BuiltinValue); ok {
		// Look up builtin implementation from registry
		impl, exists := vm.BuiltinRegistry[builtinVal.Name]
		if !exists {
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

		// Call the builtin implementation
		result, err := impl(args)
		if err != nil {
			return nil, err
		}

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
		return nil, fmt.Errorf("Compiler error: stack contains non-callable value on call")
	}
	ptr := vm.ExecPtr(fnPtr)
	args := make([]vm.ArgValue, n)
	for i := n - 1; i >= 0; i-- {
		args[i], ok = frame.Pop().(vm.ArgValue)
		if !ok {
			return nil, fmt.Errorf("Compiler error: stack contains non-call arguments")
		}
	}
	newFrame := &StackFrame{
		PC: ptr,
	}
	fn := prog.Code[ptr.CodeID()-1]
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
			return nil, fmt.Errorf("Not enough arguments to call")
		}
	}
	return newFrame, nil
}
