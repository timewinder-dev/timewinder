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
	Stacks      []StackFrames // Multiple threads in this set
	PauseReason []Pause       // Parallel to Stacks
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
	Yield
	NonDet            // Paused due to non-deterministic value (oneof)
	WeaklyFairYield   // Weakly fair yield (from fstep) - no stutter checking
	Stuttering        // Virtual state for stutter checking (as if process terminates)
	Waiting           // Blocked on until() condition
	WeaklyFairWaiting // Blocked on funtil() condition
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
	case WeaklyFairYield:
		return "WeaklyFairYield"
	case Stuttering:
		return "Stuttering"
	case Waiting:
		return "Waiting"
	case WeaklyFairWaiting:
		return "WeaklyFairWaiting"
	default:
		return fmt.Sprintf("Unknown(%d)", p)
	}
}
