package model

import (
	"fmt"

	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

func RunTrace(t *Thunk, prog *vm.Program) (*interp.State, []vm.Value, error) {
	state := t.State.Clone()
	choices, err := interp.RunToPause(prog, state, t.ToRun)
	if err != nil {
		return nil, nil, err
	}
	return state, choices, nil
}

func BuildRunnable(t *Thunk, state *interp.State, exec *Executor) ([]*Thunk, error) {
	// Hash the state (CAS handles decomposition internally)
	stateHash, err := exec.CAS.Put(state)
	if err != nil {
		return nil, fmt.Errorf("hashing state: %w", err)
	}

	// Check if we've visited this state (cycle detection)
	if exec.VisitedStates[stateHash] {
		return nil, nil // Prune: already explored
	}
	exec.VisitedStates[stateHash] = true

	var out []*Thunk
	n := t.Clone()
	states, err := interp.Canonicalize(state)
	if err != nil {
		return nil, err
	}
	trace := TraceStep{ThreadRan: t.ToRun, StateHash: stateHash}
	n.Trace = append(n.Trace, trace)
	for i, r := range state.PauseReason {
		switch r {
		case interp.Finished:
			continue
		case interp.Yield:
			fallthrough
		case interp.Start:
			for _, s := range states {
				x := n.Clone()
				x.ToRun = i
				x.State = s
				out = append(out, x)
			}
		}
	}
	return out, nil
}

func CheckProperties(state *interp.State, props []Property) error {
	for _, prop := range props {
		result, err := prop.Check(state)
		if err != nil {
			// Execution error - propagate it
			return fmt.Errorf("Error checking property: %w", err)
		}
		if !result.Success {
			// Property violation - return error with details
			return fmt.Errorf("Property violation: %s", result.Message)
		}
	}
	return nil
}
