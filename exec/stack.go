package exec

import (
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/vm"
)

type Stack struct {
	Frames []Frame
}

type Frame struct {
	Entries map[string]vm.Value
}

type StackHash cas.Hash
