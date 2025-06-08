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
		return nil, fmt.Errorf("Call stack is too short to buildCallFrame")
	}
	fnPtr, ok := frame.Pop().(vm.FnPtrValue)
	if !ok {
		return nil, fmt.Errorf("Compiler error: stack contains non-Fn-Ptr on call")
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
