package model

import (
	"slices"

	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
)

type Thunk struct {
	ToRun int
	State *interp.State
	Trace []TraceStep
}

func (t Thunk) Clone() *Thunk {
	return &Thunk{
		ToRun: t.ToRun,
		State: t.State,
		Trace: slices.Clone(t.Trace),
	}
}

type TraceStep struct {
	ThreadRan int
	StateHash cas.Hash
}

func (ts TraceStep) Clone() TraceStep {
	return TraceStep{
		ThreadRan: ts.ThreadRan,
		StateHash: ts.StateHash,
	}
}
