package model

import (
	"fmt"
	"io"
)

// Reporter handles progress reporting during model checking
type Reporter interface {
	Printf(format string, args ...interface{})
}

// SilentReporter does not output any progress
type SilentReporter struct{}

func (r *SilentReporter) Printf(format string, args ...interface{}) {}

// ColorReporter outputs colorized progress to a writer (typically stderr)
type ColorReporter struct {
	Writer io.Writer
}

func (r *ColorReporter) Printf(format string, args ...interface{}) {
	fmt.Fprintf(r.Writer, format, args...)
}
