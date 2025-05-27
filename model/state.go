package model

import (
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
)

type Thunk struct {
	ToRun int
	State *interp.State
	Trace []TraceStep
}

type TraceStep struct {
	ThreadRan int
	StateHash cas.Hash
}
