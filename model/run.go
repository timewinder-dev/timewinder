package model

import (
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

func RunTrace(t *Thunk, prog *vm.Program) (*interp.State, error) {

}

func BuildRunnable(t *Thunk, state *interp.State, lastState cas.Hash) ([]*Thunk, error) {

}

func CheckProperties(state *interp.State, props []Property) error {

}
