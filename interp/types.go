package interp

import "github.com/timewinder-dev/timewinder/vm"

type State struct {
	Globals *StackFrame
	Stacks  [][]*StackFrame
}

type StackFrame struct {
	Stack         []vm.Value
	PC            vm.ExecPtr
	Variables     map[string]vm.Value
	IteratorStack []*IteratorState
	PauseReason   Pause
}

type IteratorState struct {
	Start vm.ExecPtr
	End   vm.ExecPtr
	Iter  Iterator
}

func (its *IteratorState) Clone() *IteratorState {
	return &IteratorState{
		Start: its.Start,
		End:   its.End,
		Iter:  its.Iter.Clone(),
	}
}

type Iterator interface {
	Clone() Iterator
	Next() bool
	Var1() vm.Value
	Var2() vm.Value
}

type SliceIterator struct {
}

type Pause int

const (
	Start Pause = iota
	Finished
	Yield
)
