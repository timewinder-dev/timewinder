package vm

import (
	"fmt"
	"slices"

	"go.starlark.net/syntax"
)

type Special string

const (
	Step   Special = "step"
	FStep  Special = "fstep"
	Until  Special = "until"
	FUntil Special = "funtil"
)

var allSpecials = []Special{
	Step,
	FStep,
	Until,
	FUntil,
}

// Specials that don't leave values on the stack (don't need POP after them)
var specialsWithoutStackResult = []Special{
	Step,
	FStep,
	Until,
	FUntil,
}

// isSpecialWithoutStackResult checks if a call expression is a special function
// that doesn't leave a value on the stack
func isSpecialWithoutStackResult(expr syntax.Expr) bool {
	callExpr, ok := expr.(*syntax.CallExpr)
	if !ok {
		return false
	}
	ident, ok := callExpr.Fn.(*syntax.Ident)
	if !ok {
		return false
	}
	specialName := Special(ident.Name)
	for _, s := range specialsWithoutStackResult {
		if specialName == s {
			return true
		}
	}
	return false
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
		cc.emit(FAIR_YIELD, StrValue(v.Value.(string)))
	case Until:
		return cc.compileUntil(call, false)
	case FUntil:
		return cc.compileUntil(call, true)
	default:
		return true, fmt.Errorf("Unhandled special: %s", fn.Name)
	}
	return true, nil
}

func (cc *compileContext) compileUntil(call *syntax.CallExpr, isWeaklyFair bool) (bool, error) {
	if len(call.Args) != 1 {
		funcName := "until"
		if isWeaklyFair {
			funcName = "funtil"
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

	// Emit conditional yield with retry label
	// Note: CONDITIONAL_YIELD/CONDITIONAL_FAIR_YIELD consume the condition value
	if isWeaklyFair {
		cc.emit(CONDITIONAL_FAIR_YIELD, StrValue(condStartLabel))
	} else {
		cc.emit(CONDITIONAL_YIELD, StrValue(condStartLabel))
	}
	// No POP needed - conditional yields already consume the condition value

	return true, nil
}
