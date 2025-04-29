package model

import (
	"github.com/timewinder-dev/timewinder/exec"
	"github.com/timewinder-dev/timewinder/vm"
)

// State represents an evaluation node of the model. It is at all times "ready
// to execute" -- ie, it contains no data about the past, only references enough to
// know where all parallel threads are and can roll execution forward.
type State struct {
	ToRun         int
	CanRun        []int
	CouldRun      map[int]Predicate
	Stacks        []exec.StackHash
	Continuations []vm.ExecPtr
}
