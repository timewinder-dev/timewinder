package interp

import (
	"errors"
	"fmt"
	"sort"

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
	NonDetStep // Non-deterministic choice encountered (from oneof builtin)
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
	if err == nil {
		fmt.Printf("DEBUG Step: PC=%v, opcode=%s, stack_len=%d\n", frame.PC, inst.Code, len(frame.Stack))
	}
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
	case vm.DUP:
		a := frame.Pop()
		frame.Push(a.Clone())
		frame.Push(a)
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
	case vm.JMP:
		// Unconditional jump to label
		if label, ok := inst.Arg.(vm.IntValue); ok {
			frame.PC = frame.PC.SetOffset(int(label))
			return ContinueStep, 0, nil
		}
		return ErrorStep, 0, fmt.Errorf("JMP requires integer label")
	case vm.JFALSE:
		// Jump to label if top of stack is false
		cond := frame.Pop()
		if !cond.AsBool() {
			if label, ok := inst.Arg.(vm.IntValue); ok {
				frame.PC = frame.PC.SetOffset(int(label))
				return ContinueStep, 0, nil
			}
			return ErrorStep, 0, fmt.Errorf("JFALSE requires integer label")
		}
		// Fall through - don't jump, just continue
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
	case vm.ITER_START:
		// Pop the iterable from stack
		iterable := frame.Pop()

		// Pop the variable name
		varName := frame.Pop()
		varNameStr := string(varName.(vm.StrValue))

		// Create appropriate iterator based on iterable type
		var iter Iterator
		switch val := iterable.(type) {
		case vm.ArrayValue:
			iter = &SliceIterator{
				Values:   val,
				Index:    -1,
				VarCount: 1,
			}
		case vm.StructValue:
			// Sort keys for deterministic iteration
			keys := make([]string, 0, len(val))
			for k := range val {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			iter = &DictIterator{
				Dict:     val,
				Keys:     keys,
				Index:    -1,
				VarCount: 1,
			}
		default:
			return ErrorStep, 0, fmt.Errorf("Cannot iterate over %T", iterable)
		}

		// Get end label from instruction arg
		endLabel := vm.ExecPtr(inst.Arg.(vm.IntValue))

		// Create and push IteratorState
		iterState := &IteratorState{
			Start:    frame.PC.Inc(), // Resume point for loop body
			End:      endLabel,        // Exit point
			Iter:     iter,
			VarNames: []string{varNameStr},
		}
		frame.IteratorStack = append(frame.IteratorStack, iterState)

		// Advance to first element
		if !iter.Next() {
			// Empty iterable, jump to end immediately
			frame.IteratorStack = frame.IteratorStack[:len(frame.IteratorStack)-1]
			frame.PC = endLabel
			return ContinueStep, 0, nil
		}

		// Set loop variable and continue to loop body (PC will auto-increment)
		frame.StoreVar(varNameStr, iter.Var1())
	case vm.ITER_START_2:
		// Pop iterable (top of stack)
		iterable := frame.Pop()

		// Pop TWO variable names (second then first - they were pushed in order var1, var2)
		varName2 := frame.Pop()  // Second variable
		varName1 := frame.Pop()  // First variable
		varName2Str := string(varName2.(vm.StrValue))
		varName1Str := string(varName1.(vm.StrValue))

		// Create iterator with VarCount=2
		var iter Iterator
		switch val := iterable.(type) {
		case vm.ArrayValue:
			iter = &SliceIterator{
				Values:   val,
				Index:    -1,
				VarCount: 2, // Index and element
			}
		case vm.StructValue:
			keys := make([]string, 0, len(val))
			for k := range val {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			iter = &DictIterator{
				Dict:     val,
				Keys:     keys,
				Index:    -1,
				VarCount: 2, // Key and value
			}
		default:
			return ErrorStep, 0, fmt.Errorf("Cannot iterate over %T", iterable)
		}

		// Same logic as ITER_START
		endLabel := vm.ExecPtr(inst.Arg.(vm.IntValue))
		iterState := &IteratorState{
			Start:    frame.PC.Inc(),
			End:      endLabel,
			Iter:     iter,
			VarNames: []string{varName1Str, varName2Str},
		}
		frame.IteratorStack = append(frame.IteratorStack, iterState)

		if !iter.Next() {
			frame.IteratorStack = frame.IteratorStack[:len(frame.IteratorStack)-1]
			frame.PC = endLabel
			return ContinueStep, 0, nil
		}

		// Set BOTH loop variables and continue to loop body (PC will auto-increment)
		frame.StoreVar(varName1Str, iter.Var1())
		frame.StoreVar(varName2Str, iter.Var2())
	case vm.ITER_NEXT:
		// Get current iterator from top of stack
		if len(frame.IteratorStack) == 0 {
			return ErrorStep, 0, fmt.Errorf("ITER_NEXT with empty iterator stack")
		}

		iterState := frame.IteratorStack[len(frame.IteratorStack)-1]
		iter := iterState.Iter

		fmt.Printf("DEBUG ITER_NEXT: calling Next(), current Index=%d\n", iter.(*DictIterator).Index)
		// Try to advance to next element
		if !iter.Next() {
			// Iterator exhausted, pop it and exit loop
			fmt.Printf("DEBUG ITER_NEXT: Iterator exhausted, jumping to End=%v\n", iterState.End)
			frame.IteratorStack = frame.IteratorStack[:len(frame.IteratorStack)-1]
			frame.PC = iterState.End
			return ContinueStep, 0, nil
		}

		fmt.Printf("DEBUG ITER_NEXT: More elements, setting vars and jumping to Start=%v\n", iterState.Start)
		// More elements, update loop variables and jump back to loop start
		if len(iterState.VarNames) == 1 {
			frame.StoreVar(iterState.VarNames[0], iter.Var1())
		} else if len(iterState.VarNames) == 2 {
			frame.StoreVar(iterState.VarNames[0], iter.Var1())
			frame.StoreVar(iterState.VarNames[1], iter.Var2())
		}
		frame.PC = iterState.Start

		return ContinueStep, 0, nil
	case vm.ITER_END:
		// Pop current iterator and jump to end
		if len(frame.IteratorStack) == 0 {
			return ErrorStep, 0, fmt.Errorf("ITER_END with empty iterator stack")
		}

		iterState := frame.IteratorStack[len(frame.IteratorStack)-1]
		frame.IteratorStack = frame.IteratorStack[:len(frame.IteratorStack)-1]
		frame.PC = iterState.End

		return ContinueStep, 0, nil
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
