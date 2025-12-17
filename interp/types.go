package interp

import (
	"fmt"
	"slices"

	"github.com/timewinder-dev/timewinder/vm"
)

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
	PendingNonDet *vm.NonDetValue // Set when a builtin returns NonDetValue
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
	Start    vm.ExecPtr
	End      vm.ExecPtr
	Iter     Iterator
	VarNames []string // Loop variable names for updating in ITER_NEXT
}

func (its *IteratorState) Clone() *IteratorState {
	return &IteratorState{
		Start:    its.Start,
		End:      its.End,
		Iter:     its.Iter.Clone(),
		VarNames: slices.Clone(its.VarNames),
	}
}

type Iterator interface {
	Clone() Iterator
	Next() bool
	Var1() vm.Value
	Var2() vm.Value
}

type Pause int

const (
	Start Pause = iota
	Finished
	Yield
	NonDet // Paused due to non-deterministic value (oneof)
)

func (p Pause) String() string {
	switch p {
	case Start:
		return "Start"
	case Finished:
		return "Finished"
	case Yield:
		return "Yield"
	case NonDet:
		return "NonDet"
	default:
		return fmt.Sprintf("Unknown(%d)", p)
	}
}
