package vm

import (
	"fmt"
	"slices"

	"go.starlark.net/syntax"
)

type Special string

const (
	Step       Special = "step"
	FStep      Special = "fstep"
	Wait       Special = "wait"       // Renamed from "until"
	FWait      Special = "fwait"      // Renamed from "funtil"
	SFStep     Special = "sfstep"     // Strongly fair step
	SFWait     Special = "sfwait"     // Renamed from "sfuntil"
	StepUntil  Special = "step_until" // New: while loop + step
	FStepUntil Special = "fstep_until" // New: while loop + fstep
	SFStepUntil Special = "sfstep_until" // New: while loop + sfstep

	// Deprecated aliases for backward compatibility
	Until   Special = "until"   // Alias for wait
	FUntil  Special = "funtil"  // Alias for fwait
	SFUntil Special = "sfuntil" // Alias for sfwait
)

type FairnessType int

const (
	NormalFairness FairnessType = iota
	WeakFairness
	StrongFairness
)

var allSpecials = []Special{
	Step,
	FStep,
	Wait,
	FWait,
	SFStep,
	SFWait,
	StepUntil,
	FStepUntil,
	SFStepUntil,
	// Deprecated aliases
	Until,
	FUntil,
	SFUntil,
}


func (cc *compileContext) specialCall(call *syntax.CallExpr) (bool, error) {
	if _, ok := call.Fn.(*syntax.Ident); !ok {
		return false, nil
	}
	fn := call.Fn.(*syntax.Ident)
	if !slices.Contains(allSpecials, Special(fn.Name)) {
		return false, nil
	}
	switch Special(fn.Name) {
	case Step:
		if len(call.Args) > 1 {
			return true, fmt.Errorf("Too many arguments to %s, must be a string", fn.Name)
		} else if len(call.Args) == 0 {
			return true, fmt.Errorf("No arguments to %s, must label the step", fn.Name)
		} else if _, ok := call.Args[0].(*syntax.Literal); !ok {
			return true, fmt.Errorf("Argument to %s is not a literal value", fn.Name)
		}
		v := call.Args[0].(*syntax.Literal)
		if v.Token != syntax.STRING {
			return true, fmt.Errorf("Argument to %s is not a literal string label", fn.Name)
		}
		// Push the step name to stack first (for ExprStmt POP to consume)
		cc.emit(PUSH, StrValue(v.Value.(string)))
		// Then yield (arg is for tracing/debugging)
		cc.emit(YIELD, StrValue(v.Value.(string)))
	case FStep:
		if len(call.Args) > 1 {
			return true, fmt.Errorf("Too many arguments to %s, must be a string", fn.Name)
		} else if len(call.Args) == 0 {
			return true, fmt.Errorf("No arguments to %s, must label the step", fn.Name)
		} else if _, ok := call.Args[0].(*syntax.Literal); !ok {
			return true, fmt.Errorf("Argument to %s is not a literal value", fn.Name)
		}
		v := call.Args[0].(*syntax.Literal)
		if v.Token != syntax.STRING {
			return true, fmt.Errorf("Argument to %s is not a literal string label", fn.Name)
		}
		// Push the step name to stack first (for ExprStmt POP to consume)
		cc.emit(PUSH, StrValue(v.Value.(string)))
		// Then yield with weak fairness (arg is for tracing/debugging)
		cc.emit(FAIR_YIELD, StrValue(v.Value.(string)))
	case SFStep:
		if len(call.Args) > 1 {
			return true, fmt.Errorf("Too many arguments to %s, must be a string", fn.Name)
		} else if len(call.Args) == 0 {
			return true, fmt.Errorf("No arguments to %s, must label the step", fn.Name)
		} else if _, ok := call.Args[0].(*syntax.Literal); !ok {
			return true, fmt.Errorf("Argument to %s is not a literal value", fn.Name)
		}
		v := call.Args[0].(*syntax.Literal)
		if v.Token != syntax.STRING {
			return true, fmt.Errorf("Argument to %s is not a literal string label", fn.Name)
		}
		// Push the step name to stack first (for ExprStmt POP to consume)
		cc.emit(PUSH, StrValue(v.Value.(string)))
		// Then yield with strong fairness (arg is for tracing/debugging)
		cc.emit(STRONG_YIELD, StrValue(v.Value.(string)))
	case Wait, Until: // wait() or until() (deprecated alias)
		return cc.compileWait(call, NormalFairness)
	case FWait, FUntil: // fwait() or funtil() (deprecated alias)
		return cc.compileWait(call, WeakFairness)
	case SFWait, SFUntil: // sfwait() or sfuntil() (deprecated alias)
		return cc.compileWait(call, StrongFairness)
	case StepUntil: // step_until(label, condition) - while condition: step(label)
		return cc.compileStepUntil(call, NormalFairness)
	case FStepUntil: // fstep_until(label, condition) - while condition: fstep(label)
		return cc.compileStepUntil(call, WeakFairness)
	case SFStepUntil: // sfstep_until(label, condition) - while condition: sfstep(label)
		return cc.compileStepUntil(call, StrongFairness)
	default:
		return true, fmt.Errorf("Unhandled special: %s", fn.Name)
	}
	return true, nil
}

func (cc *compileContext) compileWait(call *syntax.CallExpr, fairness FairnessType) (bool, error) {
	if len(call.Args) != 1 {
		funcName := "wait"
		switch fairness {
		case WeakFairness:
			funcName = "fwait"
		case StrongFairness:
			funcName = "sfwait"
		}
		return true, fmt.Errorf("%s takes exactly 1 argument (condition expression), got %d", funcName, len(call.Args))
	}

	// Mark the PC where condition starts (for re-evaluation)
	condStartLabel := cc.newLabel()
	cc.emitLabel(condStartLabel)

	// Compile condition expression
	err := cc.expr(call.Args[0])
	if err != nil {
		return true, err
	}

	// Emit conditional yield with retry label based on fairness type
	// Note: CONDITIONAL_* opcodes consume the condition value
	switch fairness {
	case NormalFairness:
		cc.emit(CONDITIONAL_YIELD, StrValue(condStartLabel))
	case WeakFairness:
		cc.emit(CONDITIONAL_FAIR_YIELD, StrValue(condStartLabel))
	case StrongFairness:
		cc.emit(CONDITIONAL_STRONG_YIELD, StrValue(condStartLabel))
	}

	// Push None to leave a value on stack (for ExprStmt POP to consume)
	cc.emit(PUSH, None)

	return true, nil
}

func (cc *compileContext) compileStepUntil(call *syntax.CallExpr, fairness FairnessType) (bool, error) {
	if len(call.Args) != 2 {
		funcName := "step_until"
		switch fairness {
		case WeakFairness:
			funcName = "fstep_until"
		case StrongFairness:
			funcName = "sfstep_until"
		}
		return true, fmt.Errorf("%s takes exactly 2 arguments (label, condition), got %d", funcName, len(call.Args))
	}

	// First argument must be a string literal (label)
	labelLit, ok := call.Args[0].(*syntax.Literal)
	if !ok || labelLit.Token != syntax.STRING {
		return true, fmt.Errorf("first argument to step_until must be a string literal label")
	}
	label := StrValue(labelLit.Value.(string))

	// Compile to: while condition: step(label)
	// Structure:
	//   loop_start:
	//     <evaluate condition>
	//     JFALSE loop_end
	//     PUSH label
	//     YIELD/FAIR_YIELD/STRONG_YIELD label
	//     JMP loop_start
	//   loop_end:
	//     PUSH None

	loopStart := cc.newLabel()
	loopEnd := cc.newLabel()

	// loop_start:
	cc.emitLabel(loopStart)

	// Evaluate condition
	err := cc.expr(call.Args[1])
	if err != nil {
		return true, err
	}

	// JFALSE loop_end (if condition is false, exit loop)
	cc.emit(JFALSE, StrValue(loopEnd))

	// Push label and yield
	cc.emit(PUSH, label)
	switch fairness {
	case NormalFairness:
		cc.emit(YIELD, label)
	case WeakFairness:
		cc.emit(FAIR_YIELD, label)
	case StrongFairness:
		cc.emit(STRONG_YIELD, label)
	}

	// POP the label from stack (cleanup after yield)
	cc.emit(POP)

	// JMP loop_start (continue loop)
	cc.emit(JMP, StrValue(loopStart))

	// loop_end:
	cc.emitLabel(loopEnd)

	// Push None to leave value on stack (for ExprStmt POP to consume)
	cc.emit(PUSH, None)

	return true, nil
}
