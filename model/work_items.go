package model

import (
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
)

// WorkItem represents a unit of work for execution workers.
// It contains a thunk (state + thread to run) and the depth at which it occurs.
type WorkItem struct {
	Thunk       *Thunk
	DepthNumber int
}

// CheckItem represents a unit of work for checking workers.
// It contains all the information needed to check properties, detect cycles, and handle violations.
type CheckItem struct {
	Thunk          *Thunk
	State          *interp.State
	StateHash      cas.Hash
	IsNewState     bool // false indicates this is a cycle (state seen before)
	DepthNumber    int
	SuccessorCount int // Number of successors generated (0 indicates terminating state)
}

// NewWorkItem creates a new WorkItem from a thunk and depth number.
func NewWorkItem(thunk *Thunk, depth int) *WorkItem {
	return &WorkItem{
		Thunk:       thunk,
		DepthNumber: depth,
	}
}

// NewCheckItem creates a new CheckItem with all necessary information for property checking.
func NewCheckItem(thunk *Thunk, state *interp.State, hash cas.Hash, isNew bool, depth int, successorCount int) *CheckItem {
	return &CheckItem{
		Thunk:          thunk,
		State:          state,
		StateHash:      hash,
		IsNewState:     isNew,
		DepthNumber:    depth,
		SuccessorCount: successorCount,
	}
}
