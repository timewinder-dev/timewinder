package interp

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/timewinder-dev/timewinder/vm"
)

type StepResult int

const (
	ContinueStep StepResult = iota
	ReturnStep
	EndStep
	CallStep
	MethodCallStep // Method call encountered (e.g., arr.append(x))
	ErrorStep
	YieldStep
	NonDetStep // Non-deterministic choice encountered (from oneof builtin)
)

// YieldType distinguishes between different types of yields
type YieldType int

const (
	YieldNormal              YieldType = iota
	YieldWeaklyFair                    // Weakly fair yield (from fstep) - no stutter checking
	YieldStronglyFair                  // Strongly fair yield (from sfstep) - no stutter checking
	YieldWaiting                       // Waiting on condition (from until)
	YieldWeaklyFairWaiting             // Weakly fair waiting (from funtil)
	YieldStronglyFairWaiting           // Strongly fair waiting (from sfuntil)
)

type Program interface {
	GetInstruction(vm.ExecPtr) (vm.Op, error)
	Resolve(name string) (vm.ExecPtr, bool)
	GetFunction(vm.ExecPtr) *vm.Function
}

func Step(program Program, globals *StackFrame, stack []*StackFrame) (StepResult, int, error) {
	if len(stack) == 0 {
		log.Trace().Msg("Step: empty stack, returning error")
		return ErrorStep, 0, errors.New("No stack frame")
	}
	frame := stack[len(stack)-1]
	inst, err := program.GetInstruction(frame.PC)
	if err != nil {
		if errors.Is(err, vm.ErrEndOfCode) {
			log.Trace().Str("pc", frame.PC.String()).Msg("Step: end of code")
			return EndStep, 0, nil
		}
		log.Trace().Err(err).Str("pc", frame.PC.String()).Msg("Step: error getting instruction")
		return ErrorStep, 0, err
	}

	log.Trace().
		Str("opcode", inst.Code.String()).
		Str("pc", frame.PC.String()).
		Interface("arg", inst.Arg).
		Int("stack_depth", len(frame.Stack)).
		Interface("stack", frame.Stack).
		Msg("Step: executing instruction")

	switch inst.Code {
	case vm.NOP:
		log.Trace().Interface("stack", frame.Stack).Msg("  NOP")
	case vm.POP:
		val := frame.Pop()
		log.Trace().Interface("value", val).Interface("stack", frame.Stack).Msg("  POP")
	case vm.PUSH:
		frame.Push(inst.Arg.Clone())
		log.Trace().Interface("value", inst.Arg).Interface("stack", frame.Stack).Msg("  PUSH")
	case vm.SETVAL:
		name := frame.Pop()
		val := frame.Pop()
		variable := mustString(name)

		// New scoping rules: unified namespace with shadowing detection
		// Check if variable exists in both globals and local scope
		var inGlobals bool
		var inLocal bool

		// Check globals
		if globals != nil {
			_, inGlobals = globals.Variables[variable]
		}

		// Check current frame (local scope)
		_, inLocal = frame.Variables[variable]

		// Shadowing detection: error if variable exists in both scopes
		if inGlobals && inLocal {
			log.Trace().Str("variable", variable).Msg("  SETVAL: shadowing detected")
			return ErrorStep, 0, fmt.Errorf("Variable shadowing detected: '%s' exists in both global and local scope", variable)
		}

		// Write to globals if variable exists there
		if inGlobals {
			globals.StoreVar(variable, val)
			log.Trace().Str("variable", variable).Interface("value", val).Str("scope", "global").Interface("stack", frame.Stack).Msg("  SETVAL")
		} else {
			// Write to local scope (create new local if needed)
			frame.StoreVar(variable, val)
			log.Trace().Str("variable", variable).Interface("value", val).Str("scope", "local").Interface("stack", frame.Stack).Msg("  SETVAL")
		}
	case vm.GETVAL:
		name := frame.Pop()
		varName := mustString(name)
		v, err := resolveVar(varName, program, globals, stack)
		if err != nil {
			log.Trace().Str("variable", varName).Err(err).Interface("stack", frame.Stack).Msg("  GETVAL: error")
			return ErrorStep, 0, err
		}
		frame.Push(v)
		log.Trace().Str("variable", varName).Interface("value", v).Interface("stack", frame.Stack).Msg("  GETVAL")
	case vm.SWAP:
		a := frame.Pop()
		b := frame.Pop()
		frame.Push(a)
		frame.Push(b)
		log.Trace().Interface("stack", frame.Stack).Msg("  SWAP")
	case vm.DUP:
		a := frame.Pop()
		frame.Push(a.Clone())
		frame.Push(a)
		log.Trace().Interface("value", a).Interface("stack", frame.Stack).Msg("  DUP")
	case vm.GETATTR:
		// Stack: A B -> C where C = A[B]
		key := frame.Pop()
		obj := frame.Pop()
		val, err := getAttribute(obj, key)
		if err != nil {
			log.Trace().Interface("obj", obj).Interface("key", key).Err(err).Msg("  GETATTR: error")
			return ErrorStep, 0, err
		}
		frame.Push(val)
		log.Trace().Interface("obj", obj).Interface("key", key).Interface("value", val).Interface("stack", frame.Stack).Msg("  GETATTR")
	case vm.SETATTR:
		// Stack: C A B -> nothing, sets A[B] = C
		key := frame.Pop()
		obj := frame.Pop()
		val := frame.Pop()
		err := setAttribute(obj, key, val)
		if err != nil {
			log.Trace().Interface("obj", obj).Interface("key", key).Interface("value", val).Err(err).Msg("  SETATTR: error")
			return ErrorStep, 0, err
		}
		log.Trace().Interface("obj", obj).Interface("key", key).Interface("value", val).Interface("stack", frame.Stack).Msg("  SETATTR")
	case vm.NOT:
		a := frame.Pop()
		new := !a.AsBool()
		frame.Push(vm.BoolValue(new))
		log.Trace().Interface("input", a).Bool("result", new).Interface("stack", frame.Stack).Msg("  NOT")
	case vm.ADD:
		b := frame.Pop()
		a := frame.Pop()
		v, err := add(a, b)
		if err != nil {
			log.Trace().Interface("a", a).Interface("b", b).Err(err).Msg("  ADD: error")
			return ErrorStep, 0, err
		}
		frame.Push(v)
		log.Trace().Interface("a", a).Interface("b", b).Interface("result", v).Interface("stack", frame.Stack).Msg("  ADD")
	case vm.MULTIPLY:
		fallthrough
	case vm.DIVIDE:
		fallthrough
	case vm.MODULO:
		fallthrough
	case vm.FLOOR_DIVIDE:
		fallthrough
	case vm.POWER:
		fallthrough
	case vm.SUBTRACT:
		b := frame.Pop()
		a := frame.Pop()
		v, err := numericOp(inst.Code, a, b)
		if err != nil {
			log.Trace().Str("op", inst.Code.String()).Interface("a", a).Interface("b", b).Err(err).Msg("  NUMERIC_OP: error")
			return ErrorStep, 0, err
		}
		frame.Push(v)
		log.Trace().Str("op", inst.Code.String()).Interface("a", a).Interface("b", b).Interface("result", v).Interface("stack", frame.Stack).Msg("  NUMERIC_OP")
	case vm.EQ:
		b := frame.Pop()
		a := frame.Pop()
		v, ok := a.Cmp(b)
		var result vm.Value
		if !ok {
			// Not comparable, therefore not equal
			result = vm.BoolFalse
			frame.Push(result)
		} else {
			if v == 0 {
				result = vm.BoolTrue
			} else {
				result = vm.BoolFalse
			}
			frame.Push(result)
		}
		log.Trace().Interface("a", a).Interface("b", b).Interface("result", result).Interface("stack", frame.Stack).Msg("  EQ")
	case vm.LT:
		b := frame.Pop()
		a := frame.Pop()
		v, ok := a.Cmp(b)
		if !ok {
			log.Trace().Interface("a", a).Interface("b", b).Msg("  LT: incomparable types")
			return ErrorStep, 0, fmt.Errorf("Can't compare %#v to %#v", a, b)
		}
		var result vm.Value
		if v < 0 {
			result = vm.BoolTrue
		} else {
			result = vm.BoolFalse
		}
		frame.Push(result)
		log.Trace().Interface("a", a).Interface("b", b).Interface("result", result).Interface("stack", frame.Stack).Msg("  LT")
	case vm.LTE:
		b := frame.Pop()
		a := frame.Pop()
		v, ok := a.Cmp(b)
		if !ok {
			log.Trace().Interface("a", a).Interface("b", b).Msg("  LTE: incomparable types")
			return ErrorStep, 0, fmt.Errorf("Can't compare %#v to %#v", a, b)
		}
		var result vm.Value
		if v <= 0 {
			result = vm.BoolTrue
		} else {
			result = vm.BoolFalse
		}
		frame.Push(result)
		log.Trace().Interface("a", a).Interface("b", b).Interface("result", result).Interface("stack", frame.Stack).Msg("  LTE")
	case vm.IN:
		// Stack: item collection -> bool (item in collection)
		collection := frame.Pop()
		item := frame.Pop()

		var result vm.Value
		switch coll := collection.(type) {
		case vm.ArrayValue:
			// Check if item is in array
			found := false
			for _, elem := range coll {
				if eq, ok := item.Cmp(elem); ok && eq == 0 {
					found = true
					break
				}
			}
			if found {
				result = vm.BoolTrue
			} else {
				result = vm.BoolFalse
			}
		case vm.StrValue:
			// Check if substring is in string
			itemStr, ok := item.(vm.StrValue)
			if !ok {
				return ErrorStep, 0, fmt.Errorf("IN operator: can only check for string in string, got %T in string", item)
			}
			if strings.Contains(string(coll), string(itemStr)) {
				result = vm.BoolTrue
			} else {
				result = vm.BoolFalse
			}
		case vm.StructValue:
			// Check if key is in dict/struct
			itemStr, ok := item.(vm.StrValue)
			if !ok {
				return ErrorStep, 0, fmt.Errorf("IN operator: can only check for string keys in struct, got %T", item)
			}
			if _, exists := coll[string(itemStr)]; exists {
				result = vm.BoolTrue
			} else {
				result = vm.BoolFalse
			}
		default:
			return ErrorStep, 0, fmt.Errorf("IN operator: unsupported collection type %T", collection)
		}
		frame.Push(result)
		log.Trace().Interface("item", item).Interface("collection", collection).Interface("result", result).Msg("  IN")
	case vm.SLICE:
		// Stack: Array Start End -> Result
		// None for start means 0, None for end means len(array)
		endVal := frame.Pop()
		startVal := frame.Pop()
		arrayVal := frame.Pop()

		arr, ok := arrayVal.(vm.ArrayValue)
		if !ok {
			return ErrorStep, 0, fmt.Errorf("SLICE requires an array, got %T", arrayVal)
		}

		// Determine start index
		start := 0
		if startVal != vm.None {
			startInt, ok := startVal.(vm.IntValue)
			if !ok {
				return ErrorStep, 0, fmt.Errorf("SLICE start index must be an integer or None, got %T", startVal)
			}
			start = int(startInt)
			if start < 0 {
				start = len(arr) + start
			}
			if start < 0 {
				start = 0
			}
			if start > len(arr) {
				start = len(arr)
			}
		}

		// Determine end index
		end := len(arr)
		if endVal != vm.None {
			endInt, ok := endVal.(vm.IntValue)
			if !ok {
				return ErrorStep, 0, fmt.Errorf("SLICE end index must be an integer or None, got %T", endVal)
			}
			end = int(endInt)
			if end < 0 {
				end = len(arr) + end
			}
			if end < 0 {
				end = 0
			}
			if end > len(arr) {
				end = len(arr)
			}
		}

		// Create sliced array
		if start > end {
			start = end
		}
		result := make(vm.ArrayValue, end-start)
		copy(result, arr[start:end])
		frame.Push(result)
	case vm.JMP:
		// Unconditional jump to label
		if label, ok := inst.Arg.(vm.IntValue); ok {
			newPC := frame.PC.SetOffset(int(label))
			log.Trace().Str("from", frame.PC.String()).Str("to", newPC.String()).Interface("stack", frame.Stack).Msg("  JMP")
			frame.PC = newPC
			return ContinueStep, 0, nil
		}
		return ErrorStep, 0, fmt.Errorf("JMP requires integer label")
	case vm.JFALSE:
		// Jump to label if top of stack is false
		cond := frame.Pop()
		condBool := cond.AsBool()
		if !condBool {
			if label, ok := inst.Arg.(vm.IntValue); ok {
				newPC := frame.PC.SetOffset(int(label))
				log.Trace().Interface("condition", cond).Bool("cond_bool", condBool).Str("from", frame.PC.String()).Str("to", newPC.String()).Interface("stack", frame.Stack).Msg("  JFALSE: jumping")
				frame.PC = newPC
				return ContinueStep, 0, nil
			}
			return ErrorStep, 0, fmt.Errorf("JFALSE requires integer label")
		}
		// Fall through - don't jump, just continue
		log.Trace().Interface("condition", cond).Bool("cond_bool", condBool).Interface("stack", frame.Stack).Msg("  JFALSE: not jumping")
	case vm.RETURN:
		log.Trace().Interface("stack", frame.Stack).Msg("  RETURN")
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
		log.Trace().Int("size", int(n)).Interface("list", l).Interface("stack", frame.Stack).Msg("  BUILD_LIST")
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
		log.Trace().Int("size", int(n)).Interface("dict", l).Interface("stack", frame.Stack).Msg("  BUILD_DICT")
	case vm.BUILD_ARG:
		name := frame.Pop()
		val := frame.Pop()
		var arg vm.ArgValue
		if _, ok := name.Cmp(vm.None); ok {
			arg = vm.ArgValue{Value: val}
			frame.Push(arg)
		} else {
			arg = vm.ArgValue{Key: mustString(name), Value: val}
			frame.Push(arg)
		}
		log.Trace().Interface("name", name).Interface("value", val).Interface("arg", arg).Interface("stack", frame.Stack).Msg("  BUILD_ARG")
	case vm.CALL:
		if v, ok := inst.Arg.(vm.IntValue); ok {
			log.Trace().Int("argc", int(v)).Interface("stack", frame.Stack).Str("pc", frame.PC.String()).Msg("  CALL")
			return CallStep, int(v), nil
		} else {
			return ErrorStep, 0, fmt.Errorf("Error in compilation; CALL should carry an int")
		}
	case vm.CALL_METHOD:
		if v, ok := inst.Arg.(vm.IntValue); ok {
			// Stack: arg1, arg2, ..., argN, receiver, methodName
			// Don't pop here - let BuildMethodCallFrame handle it
			log.Trace().Int("argc", int(v)).Interface("stack", frame.Stack).Str("pc", frame.PC.String()).Msg("  CALL_METHOD")
			return MethodCallStep, int(v), nil
		} else {
			return ErrorStep, 0, fmt.Errorf("Error in compilation; CALL_METHOD should carry an int")
		}
	case vm.YIELD:
		// Yield execution to allow other threads to run
		// The step name (inst.Arg) should already be on the stack from compilation
		// (step() compiles to PUSH <name>, YIELD <name>)
		newPC := frame.PC.Inc()
		log.Trace().Str("pc", frame.PC.String()).Str("new_pc", newPC.String()).Interface("stack", frame.Stack).Msg("  YIELD: yielding to scheduler")
		frame.PC = newPC
		return YieldStep, int(YieldNormal), nil
	case vm.FAIR_YIELD:
		// Weakly fair yield - similar to YIELD but marks as WeaklyFairYield
		// This prevents stutter checking at this point
		// The step name should already be on the stack from compilation
		newPC := frame.PC.Inc()
		log.Trace().Str("pc", frame.PC.String()).Str("new_pc", newPC.String()).Interface("stack", frame.Stack).Msg("  FAIR_YIELD: yielding (weakly fair)")
		frame.PC = newPC
		return YieldStep, int(YieldWeaklyFair), nil
	case vm.STRONG_YIELD:
		// Strongly fair yield - similar to YIELD but marks as StronglyFairYield
		// This prevents stutter checking and enforces strong fairness
		// The step name should already be on the stack from compilation
		newPC := frame.PC.Inc()
		log.Trace().Str("pc", frame.PC.String()).Str("new_pc", newPC.String()).Interface("stack", frame.Stack).Msg("  STRONG_YIELD: yielding (strongly fair)")
		frame.PC = newPC
		return YieldStep, int(YieldStronglyFair), nil
	case vm.CONDITIONAL_YIELD:
		// Pop condition result from stack
		condResult := frame.Pop()
		retryOffset, ok := inst.Arg.(vm.IntValue)
		if !ok {
			return ErrorStep, 0, fmt.Errorf("CONDITIONAL_YIELD requires integer offset")
		}

		newPC := frame.PC.Inc()
		frame.PC = newPC

		if condResult.AsBool() {
			// Condition satisfied - ALWAYS yield (to allow interleaving)
			// but thread is immediately runnable
			frame.WaitCondition = nil
			log.Trace().Bool("condition", true).Str("pc", newPC.String()).Interface("stack", frame.Stack).Msg("  CONDITIONAL_YIELD: condition satisfied, continuing")
			return ContinueStep, 0, nil // Continue - don't yield
		}

		// Condition false - yield and mark as waiting
		// Thread won't be runnable until condition becomes true
		condPC := frame.PC.SetOffset(int(retryOffset))
		frame.WaitCondition = &WaitConditionInfo{
			ConditionPC:  condPC,
			IsWeaklyFair: false,
		}
		log.Trace().Bool("condition", false).Str("pc", newPC.String()).Str("retry_pc", condPC.String()).Interface("stack", frame.Stack).Msg("  CONDITIONAL_YIELD: waiting for condition")
		return YieldStep, int(YieldWaiting), nil
	case vm.CONDITIONAL_FAIR_YIELD:
		// Pop condition result from stack
		condResult := frame.Pop()
		retryOffset, ok := inst.Arg.(vm.IntValue)
		if !ok {
			return ErrorStep, 0, fmt.Errorf("CONDITIONAL_FAIR_YIELD requires integer offset")
		}

		newPC := frame.PC.Inc()
		frame.PC = newPC

		if condResult.AsBool() {
			// Condition satisfied - continue atomically (no interleaving)
			// Weakly fair semantics (no stutter checking) but same atomicity as until()
			frame.WaitCondition = nil
			log.Trace().Bool("condition", true).Str("pc", newPC.String()).Interface("stack", frame.Stack).Msg("  CONDITIONAL_FAIR_YIELD: condition satisfied, continuing (weakly fair, atomic)")
			return ContinueStep, 0, nil // Continue atomically
		}

		// Condition false - yield and mark as weakly fair waiting
		// Thread won't be runnable until condition becomes true
		condPC := frame.PC.SetOffset(int(retryOffset))
		frame.WaitCondition = &WaitConditionInfo{
			ConditionPC:  condPC,
			IsWeaklyFair: true,
		}
		log.Trace().Bool("condition", false).Str("pc", newPC.String()).Str("retry_pc", condPC.String()).Interface("stack", frame.Stack).Msg("  CONDITIONAL_FAIR_YIELD: waiting for condition (weakly fair)")
		return YieldStep, int(YieldWeaklyFairWaiting), nil
	case vm.CONDITIONAL_STRONG_YIELD:
		// Pop condition result from stack
		condResult := frame.Pop()
		retryOffset, ok := inst.Arg.(vm.IntValue)
		if !ok {
			return ErrorStep, 0, fmt.Errorf("CONDITIONAL_STRONG_YIELD requires integer offset")
		}

		newPC := frame.PC.Inc()
		frame.PC = newPC

		if condResult.AsBool() {
			// Condition satisfied - continue atomically (no interleaving)
			// Strongly fair semantics (no stutter checking) but same atomicity as until()
			frame.WaitCondition = nil
			log.Trace().Bool("condition", true).Str("pc", newPC.String()).Interface("stack", frame.Stack).Msg("  CONDITIONAL_STRONG_YIELD: condition satisfied, continuing (strongly fair, atomic)")
			return ContinueStep, 0, nil // Continue atomically
		}

		// Condition false - yield and mark as strongly fair waiting
		// Thread won't be runnable until condition becomes true
		condPC := frame.PC.SetOffset(int(retryOffset))
		frame.WaitCondition = &WaitConditionInfo{
			ConditionPC:    condPC,
			IsWeaklyFair:   false,
			IsStronglyFair: true,
		}
		log.Trace().Bool("condition", false).Str("pc", newPC.String()).Str("retry_pc", condPC.String()).Interface("stack", frame.Stack).Msg("  CONDITIONAL_STRONG_YIELD: waiting for condition (strongly fair)")
		return YieldStep, int(YieldStronglyFairWaiting), nil
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
			log.Trace().Interface("iterable", iterable).Msg("  ITER_START: cannot iterate")
			return ErrorStep, 0, fmt.Errorf("Cannot iterate over %T", iterable)
		}

		// Get end label from instruction arg (preserve CodeID, set offset)
		endLabel := frame.PC.SetOffset(int(inst.Arg.(vm.IntValue)))

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
			log.Trace().Str("var", varNameStr).Interface("iterable", iterable).Str("end_pc", endLabel.String()).Msg("  ITER_START: empty iterable, jumping to end")
			return ContinueStep, 0, nil
		}

		// Set loop variable and continue to loop body (PC will auto-increment)
		firstVal := iter.Var1()
		frame.StoreVar(varNameStr, firstVal)
		log.Trace().Str("var", varNameStr).Interface("iterable", iterable).Interface("first_value", firstVal).Str("start_pc", iterState.Start.String()).Str("end_pc", endLabel.String()).Msg("  ITER_START: starting iteration")
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
		endLabel := frame.PC.SetOffset(int(inst.Arg.(vm.IntValue)))
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
			log.Trace().Msg("  ITER_NEXT: empty iterator stack")
			return ErrorStep, 0, fmt.Errorf("ITER_NEXT with empty iterator stack")
		}

		iterState := frame.IteratorStack[len(frame.IteratorStack)-1]
		iter := iterState.Iter

		// Try to advance to next element
		if !iter.Next() {
			// Iterator exhausted, pop it and exit loop
			frame.IteratorStack = frame.IteratorStack[:len(frame.IteratorStack)-1]
			frame.PC = iterState.End
			log.Trace().Str("end_pc", iterState.End.String()).Msg("  ITER_NEXT: exhausted, exiting loop")
			return ContinueStep, 0, nil
		}
		// More elements, update loop variables and jump back to loop start
		if len(iterState.VarNames) == 1 {
			val := iter.Var1()
			frame.StoreVar(iterState.VarNames[0], val)
			log.Trace().Str("var", iterState.VarNames[0]).Interface("value", val).Str("start_pc", iterState.Start.String()).Msg("  ITER_NEXT: continuing iteration")
		} else if len(iterState.VarNames) == 2 {
			val1 := iter.Var1()
			val2 := iter.Var2()
			frame.StoreVar(iterState.VarNames[0], val1)
			frame.StoreVar(iterState.VarNames[1], val2)
			log.Trace().Str("var1", iterState.VarNames[0]).Interface("value1", val1).Str("var2", iterState.VarNames[1]).Interface("value2", val2).Str("start_pc", iterState.Start.String()).Msg("  ITER_NEXT: continuing iteration")
		}
		frame.PC = iterState.Start

		return ContinueStep, 0, nil
	case vm.ITER_END:
		// Pop current iterator and jump to end
		if len(frame.IteratorStack) == 0 {
			log.Trace().Msg("  ITER_END: empty iterator stack")
			return ErrorStep, 0, fmt.Errorf("ITER_END with empty iterator stack")
		}

		iterState := frame.IteratorStack[len(frame.IteratorStack)-1]
		frame.IteratorStack = frame.IteratorStack[:len(frame.IteratorStack)-1]
		frame.PC = iterState.End
		log.Trace().Str("end_pc", iterState.End.String()).Msg("  ITER_END: breaking from loop")

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
	case vm.MODULO:
		return vm.FloatValue(math.Mod(a, b))
	case vm.FLOOR_DIVIDE:
		return vm.FloatValue(math.Floor(a / b))
	case vm.POWER:
		return vm.FloatValue(math.Pow(a, b))
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
	case vm.MODULO:
		return vm.IntValue(a % b)
	case vm.FLOOR_DIVIDE:
		return vm.IntValue(a / b) // Integer division is already floor division in Go
	case vm.POWER:
		return vm.IntValue(int(math.Pow(float64(a), float64(b))))
	}
	panic("Unhandled intOp code")
}

func mustString(v vm.Value) string {
	return string(v.(vm.StrValue))
}

func resolveVar(name string, program Program, globals *StackFrame, stack []*StackFrame) (vm.Value, error) {
	// New scoping rules: unified namespace with shadowing detection
	// Check if variable exists in both globals and local scope (current frame)
	var inGlobals bool
	var inLocal bool
	var globalVal vm.Value
	var localVal vm.Value

	// Check globals
	if globals != nil {
		if v, ok := globals.Variables[name]; ok {
			inGlobals = true
			globalVal = v
		}
	}

	// Check current frame (local scope)
	if len(stack) > 0 {
		currentFrame := stack[len(stack)-1]
		if v, ok := currentFrame.Variables[name]; ok {
			inLocal = true
			localVal = v
		}
	}

	// Shadowing detection: error if variable exists in both scopes
	if inGlobals && inLocal {
		return nil, fmt.Errorf("Variable shadowing detected: '%s' exists in both global and local scope", name)
	}

	// Return from globals first (if exists)
	if inGlobals {
		return globalVal, nil
	}

	// Return from local scope (if exists)
	if inLocal {
		return localVal, nil
	}

	// Check if it's a function name
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
