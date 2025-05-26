package vm

import (
	"fmt"
	"slices"

	"go.starlark.net/syntax"
)

type Special string

const (
	Step Special = "step"
)

var allSpecials = []Special{
	Step,
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
	default:
		return true, fmt.Errorf("Unhandled special: %s", fn.Name)
	}
	return true, nil
}
