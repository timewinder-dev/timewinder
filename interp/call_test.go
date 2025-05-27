package interp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/timewinder-dev/timewinder/vm"
)

var code = `
def someArgs(x, y, z=3):
	return x + y + z
`

func TestFunctionCall(t *testing.T) {
	prg, err := vm.CompileLiteral(code)
	prg.DebugPrint()
	require.NoError(t, err)
	_, err = FunctionCallFromString(prg, &StackFrame{}, "someArgs()")
	require.Error(t, err)
	_, err = FunctionCallFromString(prg, &StackFrame{}, "someArgs(1)")
	require.Error(t, err)
	_, err = FunctionCallFromString(prg, &StackFrame{}, "someArgs(1, 2)")
	require.NoError(t, err)
	_, err = FunctionCallFromString(prg, &StackFrame{}, "someArgs(1, 2, 3)")
	require.NoError(t, err)
	_, err = FunctionCallFromString(prg, &StackFrame{}, "someArgs(y=1, x=2)")
	require.NoError(t, err)
	_, err = FunctionCallFromString(prg, &StackFrame{}, "someArgs(y=1, x=2, 2)")
	require.NoError(t, err)
}
