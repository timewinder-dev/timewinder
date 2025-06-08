package interp

import "github.com/timewinder-dev/timewinder/vm"

type State struct {
	Globals     *StackFrame
	Stacks      []StackFrames
	PauseReason []Pause
}

type StackFrame struct {
	Stack         []vm.Value
	PC            vm.ExecPtr
	Variables     map[string]vm.Value
	IteratorStack []*IteratorState
}

type StackFrames []*StackFrame

func (s *StackFrames) PopStack() *StackFrame {
	f := s.CurrentStack()
	*s = (*s)[:len(*s)-1]
	return f
}

func (s *StackFrames) Append(f *StackFrame) {
	*s = append(*s, f)
}

func (s StackFrames) CurrentStack() *StackFrame {
	return s[len(s)-1]
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
