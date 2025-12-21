package interp

import (
	"fmt"
	"slices"

	"github.com/timewinder-dev/timewinder/vm"
)

// ThreadID identifies a specific thread by its ThreadSet index and local index within that set
type ThreadID struct {
	SetIdx   int // Index of the ThreadSet
	LocalIdx int // Index within the ThreadSet
}

// ThreadSet groups symmetric (interchangeable) threads together
type ThreadSet struct {
	Stacks       []StackFrames // Multiple threads in this set
	PauseReason  []Pause       // Parallel to Stacks
	WeaklyFair   []bool        // Parallel to Stacks - true if last yield was from fstep() (weakly fair)
}

type State struct {
	Globals    *StackFrame
	ThreadSets []ThreadSet // Groups of symmetric threads
}

type StackFrame struct {
	Stack          []vm.Value
	PC             vm.ExecPtr
	Variables      map[string]vm.Value
	IteratorStack  []*IteratorState
	PendingNonDet  *vm.NonDetValue     // Set when a builtin returns NonDetValue
	WaitCondition  *WaitConditionInfo  // Set when thread is waiting on until()/funtil()
}

type WaitConditionInfo struct {
	ConditionPC  vm.ExecPtr // PC pointing to start of condition expression
	IsWeaklyFair bool       // true for funtil(), false for until()
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
	Runnable   // Thread can run (from step() or fstep())
	NonDet     // Paused due to non-deterministic value (oneof)
	Blocked    // Thread is blocked on condition (from until() or funtil())
	Stuttering // Virtual state for stutter checking (as if process terminates)
)

func (p Pause) String() string {
	switch p {
	case Start:
		return "Start"
	case Finished:
		return "Finished"
	case Runnable:
		return "Runnable"
	case NonDet:
		return "NonDet"
	case Blocked:
		return "Blocked"
	case Stuttering:
		return "Stuttering"
	default:
		return fmt.Sprintf("Unknown(%d)", p)
	}
}
