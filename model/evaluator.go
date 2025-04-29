package model

import "github.com/timewinder-dev/timewinder/vm"

// The role of an evaluator is to take a State and compare it to the
// provided invariants, or optionally, take a trace and check against the liveness
// properties.
//
// An Evaluator is intended to be stateless, writing to a CAS and initialized with the
// predicates.
type Evaluator struct {
}

type Predicate interface {
	Evaluate(stack Stack) (bool, error)
}

var _ Predicate = &ExecPtrPredicate{}

type ExecPtrPredicate struct {
	Ptr vm.ExecPtr
}
