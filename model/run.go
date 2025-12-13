package model

import (
	"fmt"

	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

func RunTrace(t *Thunk, prog *vm.Program) (*interp.State, error) {
	state := t.State.Clone()
	_, err := interp.RunToPause(prog, state, t.ToRun)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func BuildRunnable(t *Thunk, state *interp.State, lastState cas.Hash) ([]*Thunk, error) {
	var out []*Thunk
	n := t.Clone()
	states, err := interp.Canonicalize(state)
	if err != nil {
		return nil, err
	}
	trace := TraceStep{ThreadRan: t.ToRun, StateHash: lastState}
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
			return err
		}
		if !result.Success {
			// Property violation - return error with details
			return fmt.Errorf("Property violation: %s", result.Message)
		}
	}
	return nil
}
