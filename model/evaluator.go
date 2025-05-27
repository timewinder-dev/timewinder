package model

import "github.com/timewinder-dev/timewinder/interp"

type Property struct {
	Name  string
	Start *interp.StackFrame
}
