package model

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/vm"
)

type Spec struct {
	Spec       SpecDetails             `toml:""`
	Threads    map[string]ThreadSpec   `toml:",omitempty"`
	Properties map[string]PropertySpec `toml:",omitempty"`
}

type SpecDetails struct {
	File           string `toml:",omitempty"`
	ExpectedError  string `toml:"expected_error,omitempty"`  // If set, expect a violation/error containing this substring
	NoDeadlocks    bool   `toml:"no_deadlocks,omitempty"`    // If true, disable deadlock detection (default: false, deadlocks are checked)
	NoTermination  bool   `toml:"no_termination,omitempty"`  // If true, disable termination checking (default: false, termination is checked)
}

type ThreadSpec struct {
	Entrypoint string `toml:"entrypoint,omitempty"`
	Replicas   int    `toml:"replicas,omitempty"` // Number of symmetric replicas (default: 1)
	Fair       bool   `toml:"fair,omitempty"`     // If true, use weakly fair semantics (step->fstep, until->funtil)
}

type PropertySpec struct {
	Always           string `toml:",omitempty"`
	Eventually       string `toml:",omitempty"`
	EventuallyAlways string `toml:",omitempty"`
	AlwaysEventually string `toml:",omitempty"`
}

func parseSpec(f io.Reader) (*Spec, error) {
	var out Spec
	_, err := toml.NewDecoder(f).Decode(&out)
	return &out, err
}

func LoadSpecFromFile(path string) (*Spec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	s, err := parseSpec(f)
	if err != nil {
		return nil, err
	}
	if s.Spec.File == "" {
		parts := strings.Split(fi.Name(), ".")
		parts = parts[:len(parts)-1]
		parts = append(parts, "star")
		s.Spec.File = strings.Join(parts, ".")
	}
	filedir := filepath.Dir(path)
	s.Spec.File = filepath.Clean(filepath.Join(filedir, s.Spec.File))
	return s, nil
}

// MatchesExpectedResult checks if the model result matches expectations
// Returns true if:
// - ExpectedError is empty and result.Success is true
// - ExpectedError is set and any violation message contains it (case insensitive)
func (s *Spec) MatchesExpectedResult(result *ModelResult) bool {
	if s.Spec.ExpectedError == "" {
		// No error expected - success means pass
		return result.Success
	}

	// Error expected - check if any violation matches
	if result.Success {
		return false // Expected error but got success
	}

	expectedLower := strings.ToLower(s.Spec.ExpectedError)
	for _, violation := range result.Violations {
		// Check property name
		if strings.Contains(strings.ToLower(violation.PropertyName), expectedLower) {
			return true
		}
		// Check message
		if strings.Contains(strings.ToLower(violation.Message), expectedLower) {
			return true
		}
		// Check property type
		if strings.Contains(strings.ToLower(violation.PropertyType), expectedLower) {
			return true
		}
	}

	return false
}

func (s *Spec) BuildExecutor(casStore cas.CAS) (*Executor, error) {
	p, err := vm.CompilePath(s.Spec.File)
	if err != nil {
		return nil, err
	}
	exec := &Executor{
		Program:       p,
		Spec:          s,
		DebugWriter:   io.Discard, // Default to silent; CLI can override
		CAS:           casStore,
		NoDeadlocks:   s.Spec.NoDeadlocks,   // Initialize from spec (CLI can override)
		NoTermination: s.Spec.NoTermination, // Initialize from spec (CLI can override)
	}

	// Build properties with temporal constraints
	for name, propSpec := range s.Properties {
		// Determine the temporal operator
		var operator TemporalOperator

		if propSpec.Always != "" {
			operator = Always
		} else if propSpec.EventuallyAlways != "" {
			operator = EventuallyAlways
		} else if propSpec.Eventually != "" {
			operator = Eventually
		} else if propSpec.AlwaysEventually != "" {
			operator = AlwaysEventually
		} else {
			return nil, fmt.Errorf("property %s has no operator specified", name)
		}

		// Create the underlying Property (boolean function)
		// The stack frame will be initialized later in initializeProperties()
		prop := &InterpProperty{
			Name:     name,
			Executor: exec,
		}
		exec.Properties = append(exec.Properties, prop)

		// Create TemporalConstraint that wraps the Property
		constraint := TemporalConstraint{
			Name:     name,
			Operator: operator,
			Property: prop,
		}
		exec.TemporalConstraints = append(exec.TemporalConstraints, constraint)
	}

	return exec, nil
}
