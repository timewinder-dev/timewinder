package interp

import (
	"errors"
	"fmt"

	"github.com/timewinder-dev/timewinder/vm"
)

type StepResult int

const (
	ContinueStep StepResult = iota
	ReturnStep
	EndStep
	CallStep
	ErrorStep
	YieldStep
)

type Program interface {
	GetInstruction(vm.ExecPtr) (vm.Op, error)
	Resolve(name string) (vm.ExecPtr, bool)
}

func Step(program Program, globals *StackFrame, stack []*StackFrame) (StepResult, int, error) {
	if len(stack) == 0 {
		return ErrorStep, 0, errors.New("No stack frame")
	}
	frame := stack[len(stack)-1]
	inst, err := program.GetInstruction(frame.PC)
	if err != nil {
		if errors.Is(err, vm.ErrEndOfCode) {
			return EndStep, 0, nil
		}
		return ErrorStep, 0, err
	}
	switch inst.Code {
	case vm.NOP:
	case vm.POP:
		frame.Pop()
	case vm.PUSH:
		frame.Push(inst.Arg.Clone())
	case vm.SETVAL:
		name := frame.Pop()
		val := frame.Pop()
		saved := false
		variable := mustString(name)
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i].Has(variable) {
				stack[i].StoreVar(variable, val)
				saved = true
				break
			}
		}
		if !saved && globals != nil && globals.Has(variable) {
			globals.StoreVar(variable, val)
			saved = true
		}
		if !saved {
			frame.StoreVar(variable, val)
		}
	case vm.GETVAL:
		name := frame.Pop()
		v, err := resolveVar(mustString(name), program, globals, stack)
		if err != nil {
			return ErrorStep, 0, err
		}
		frame.Push(v)
	case vm.SWAP:
		a := frame.Pop()
		b := frame.Pop()
		frame.Push(a)
		frame.Push(b)
	case vm.GETATTR:
		// Stack: A B -> C where C = A[B]
		key := frame.Pop()
		obj := frame.Pop()
		val, err := getAttribute(obj, key)
		if err != nil {
			return ErrorStep, 0, err
		}
		frame.Push(val)
	case vm.SETATTR:
		// Stack: C A B -> nothing, sets A[B] = C
		key := frame.Pop()
		obj := frame.Pop()
		val := frame.Pop()
		err := setAttribute(obj, key, val)
		if err != nil {
			return ErrorStep, 0, err
		}
	case vm.NOT:
		a := frame.Pop()
		new := !a.AsBool()
		frame.Push(vm.BoolValue(new))
	case vm.ADD:
		b := frame.Pop()
		a := frame.Pop()
		v, err := add(a, b)
		if err != nil {
			return ErrorStep, 0, err
		}
		frame.Push(v)
	case vm.MULTIPLY:
		fallthrough
	case vm.DIVIDE:
		fallthrough
	case vm.SUBTRACT:
		b := frame.Pop()
		a := frame.Pop()
		v, err := numericOp(inst.Code, a, b)
		if err != nil {
			return ErrorStep, 0, err
		}
		frame.Push(v)
	case vm.EQ:
		b := frame.Pop()
		a := frame.Pop()
		v, ok := a.Cmp(b)
		if !ok {
			// Not comparable, therefore not equal
			frame.Push(vm.BoolFalse)
		} else {
			if v == 0 {
				frame.Push(vm.BoolTrue)
			} else {
				frame.Push(vm.BoolFalse)
			}
		}
	case vm.LT:
		b := frame.Pop()
		a := frame.Pop()
		v, ok := a.Cmp(b)
		if !ok {
			return ErrorStep, 0, fmt.Errorf("Can't compare %#v to %#v", a, b)
		}
		if v < 0 {
			frame.Push(vm.BoolTrue)
		} else {
			frame.Push(vm.BoolFalse)
		}
	case vm.LTE:
		b := frame.Pop()
		a := frame.Pop()
		v, ok := a.Cmp(b)
		if !ok {
			return ErrorStep, 0, fmt.Errorf("Can't compare %#v to %#v", a, b)
		}
		if v <= 0 {
			frame.Push(vm.BoolTrue)
		} else {
			frame.Push(vm.BoolFalse)
		}
	case vm.RETURN:
		return ReturnStep, 0, nil
	case vm.BUILD_LIST:
		n, ok := inst.Arg.(vm.IntValue)
		if !ok {
			return ErrorStep, 0, fmt.Errorf("Error in compilation; BUILD_LIST should carry an int")
		}
		l := make([]vm.Value, int(n))
		for i := int(n) - 1; i >= 0; i-- {
			l[i] = frame.Pop()
		}
		frame.Push(vm.ArrayValue(l))
	case vm.BUILD_DICT:
		n, ok := inst.Arg.(vm.IntValue)
		if !ok {
			return ErrorStep, 0, fmt.Errorf("Error in compilation; BUILD_DICT should carry an int")
		}
		l := make(map[string]vm.Value)
		for range int(n) {
			v := frame.Pop()
			if pair, ok := v.(vm.ArrayValue); ok {
				if len(pair) != 2 {
					return ErrorStep, 0, fmt.Errorf("Error in compilation; BUILD_DICT expects pairs, not arrays")
				}
				l[mustString(pair[0])] = pair[1]
			} else {
				return ErrorStep, 0, fmt.Errorf("Error in compilation; BUILD_DICT expects pairs")
			}
		}
		frame.Push(vm.StructValue(l))
	case vm.BUILD_ARG:
		name := frame.Pop()
		val := frame.Pop()
		if _, ok := name.Cmp(vm.None); ok {
			frame.Push(vm.ArgValue{Value: val})
		} else {
			frame.Push(vm.ArgValue{Key: mustString(name), Value: val})
		}
	case vm.CALL:
		if v, ok := inst.Arg.(vm.IntValue); ok {
			return CallStep, int(v), nil
		} else {
			return ErrorStep, 0, fmt.Errorf("Error in compilation; CALL should carry an int")
		}
	case vm.YIELD:
		// Yield execution to allow other threads to run
		// Push the step name as the "return value" of step()
		// This documents which step caused the yield
		frame.Push(inst.Arg)
		frame.PC = frame.PC.Inc()
		return YieldStep, 0, nil
	default:
		return ErrorStep, 0, fmt.Errorf("Unhandled step instruction %s", inst.Code)
	}
	frame.PC = frame.PC.Inc()
	return ContinueStep, 0, nil
}

func add(a, b vm.Value) (vm.Value, error) {
	if _, ok := a.(vm.IntValue); ok {
		return numericOp(vm.ADD, a, b)
	} else if _, ok := a.(vm.FloatValue); ok {
		return numericOp(vm.ADD, a, b)
	}
	if av, ok := a.(vm.StrValue); ok {
		if bv, ok := b.(vm.StrValue); ok {
			return vm.StrValue(string(av) + string(bv)), nil
		}
	}
	return nil, fmt.Errorf("Trying to add two disparate types: %T + %T", a, b)
}

func numericOp(op vm.Opcode, a, b vm.Value) (vm.Value, error) {
	if av, ok := a.(vm.FloatValue); ok {
		if bv, ok := b.(vm.FloatValue); ok {
			return floatOp(op, float64(av), float64(bv)), nil
		} else if bv, ok := b.(vm.IntValue); ok {
			return floatOp(op, float64(av), float64(bv)), nil
		}
		return nil, fmt.Errorf("Trying to do a numeric operation between a %T and a %T", a, b)
	}
	if av, ok := a.(vm.IntValue); ok {
		if bv, ok := b.(vm.FloatValue); ok {
			return floatOp(op, float64(av), float64(bv)), nil
		} else if bv, ok := b.(vm.IntValue); ok {
			return intOp(op, int(av), int(bv)), nil
		}
		return nil, fmt.Errorf("Trying to do a numeric operation between a %T and a %T", a, b)
	}
	return nil, fmt.Errorf("Trying to do a numeric operation between a %T and a %T", a, b)
}

func floatOp(op vm.Opcode, a, b float64) vm.Value {
	switch op {
	case vm.ADD:
		return vm.FloatValue(a + b)
	case vm.SUBTRACT:
		return vm.FloatValue(a - b)
	case vm.MULTIPLY:
		return vm.FloatValue(a * b)
	case vm.DIVIDE:
		return vm.FloatValue(a / b)
	}
	panic("Unhandled floatOp code")
}

func intOp(op vm.Opcode, a, b int) vm.Value {
	switch op {
	case vm.ADD:
		return vm.IntValue(a + b)
	case vm.SUBTRACT:
		return vm.IntValue(a - b)
	case vm.MULTIPLY:
		return vm.IntValue(a * b)
	case vm.DIVIDE:
		return vm.IntValue(a / b)
	}
	panic("Unhandled intOp code")
}

func mustString(v vm.Value) string {
	return string(v.(vm.StrValue))
}

func resolveVar(name string, program Program, globals *StackFrame, stack []*StackFrame) (vm.Value, error) {
	for i := len(stack) - 1; i >= 0; i-- {
		f := stack[i]
		if v, ok := f.Variables[name]; ok {
			return v, nil
		}
	}
	if globals != nil {
		if v, ok := globals.Variables[name]; ok {
			return v, nil
		}
	}
	if v, ok := program.Resolve(name); ok {
		return vm.FnPtrValue(v), nil
	}
	return nil, fmt.Errorf("No such variable defined: %s", name)
}

func getAttribute(obj, key vm.Value) (vm.Value, error) {
	switch o := obj.(type) {
	case vm.StructValue:
		// Dictionary access
		k := mustString(key)
		if val, ok := o[k]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("Key %s not found in struct", k)
	case vm.ArrayValue:
		// Array/list access
		if idx, ok := key.(vm.IntValue); ok {
			i := int(idx)
			if i < 0 || i >= len(o) {
				return nil, fmt.Errorf("Index %d out of bounds for array of length %d", i, len(o))
			}
			return o[i], nil
		}
		return nil, fmt.Errorf("Array index must be an integer, got %T", key)
	default:
		return nil, fmt.Errorf("Cannot get attribute on type %T", obj)
	}
}

func setAttribute(obj, key, val vm.Value) error {
	switch o := obj.(type) {
	case vm.StructValue:
		// Dictionary assignment
		k := mustString(key)
		o[k] = val
		return nil
	case vm.ArrayValue:
		// Array/list assignment
		if idx, ok := key.(vm.IntValue); ok {
			i := int(idx)
			if i < 0 || i >= len(o) {
				return fmt.Errorf("Index %d out of bounds for array of length %d", i, len(o))
			}
			o[i] = val
			return nil
		}
		return fmt.Errorf("Array index must be an integer, got %T", key)
	default:
		return fmt.Errorf("Cannot set attribute on type %T", obj)
	}
}
