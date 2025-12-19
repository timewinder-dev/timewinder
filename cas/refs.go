package cas

import (
	"io"

	"github.com/shamaton/msgpack/v2"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// Thresholds for when to use hash references vs inline storage
const (
	MinStructSizeForRef = 3 // Structs with 3+ fields use hash references
	MinArraySizeForRef  = 5 // Arrays with 5+ elements use hash references
)

// ThreadInstanceRef represents a single thread within a ThreadSet
type ThreadInstanceRef struct {
	StacksHashes []Hash        // Hash for each StackFrame in this thread's call stack
	PauseReason  interp.Pause  // Why this thread is paused
}

func (t *ThreadInstanceRef) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, t)
}

func (t *ThreadInstanceRef) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, t)
}

// ThreadSetRef represents a set of symmetric (interchangeable) threads
// Threads within the set are SORTED by hash for canonicalization
type ThreadSetRef struct {
	Threads []ThreadInstanceRef // Threads in this set (sorted by hash)
}

func (t *ThreadSetRef) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, t)
}

func (t *ThreadSetRef) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, t)
}

// StateRef is the internal CAS representation of interp.State
// Instead of storing nested structures inline, it stores hashes that reference them
type StateRef struct {
	GlobalsHash Hash           // Hash of the global StackFrame
	ThreadSets  []ThreadSetRef // Groups of symmetric threads
}

func (s *StateRef) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, s)
}

func (s *StateRef) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, s)
}

// StackFrameRef is the internal CAS representation of interp.StackFrame
type StackFrameRef struct {
	StackHashes    []Hash     // Hash for each Value on the operand stack
	PC             vm.ExecPtr // Program counter (stored inline, small value)
	VariableNames  []string   // Variable names (parallel to VariableHashes)
	VariableHashes []Hash     // Variable value hashes (parallel to VariableNames)
	IteratorHashes []Hash     // Hash for each IteratorState
}

func (s *StackFrameRef) Serialize(w io.Writer) error {
	// Lists are naturally deterministic, no special handling needed
	return msgpack.MarshalWrite(w, s)
}

func (s *StackFrameRef) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, s)
}

// IteratorStateRef is the internal CAS representation of interp.IteratorState
type IteratorStateRef struct {
	Start    vm.ExecPtr // Jump target for loop start
	End      vm.ExecPtr // Jump target for loop end
	IterHash Hash       // Hash of the Iterator
	VarNames []string   // Loop variable names
}

func (s *IteratorStateRef) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, s)
}

func (s *IteratorStateRef) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, s)
}

// SliceIteratorData stores SliceIterator state for CAS
type SliceIteratorData struct {
	ValueHashes []Hash // Hashes of each element in Values
	Index       int    // Current position in iteration
	VarCount    int    // 1 or 2 variables
}

func (s *SliceIteratorData) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, s)
}

func (s *SliceIteratorData) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, s)
}

// DictIteratorData stores DictIterator state for CAS
type DictIteratorData struct {
	DictHash Hash     // Hash of the Dict StructValue
	Keys     []string // Sorted keys
	Index    int      // Current position in iteration
	VarCount int      // Should be 2 (key, value)
}

func (d *DictIteratorData) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, d)
}

func (d *DictIteratorData) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, d)
}

// StructValueRef is used for large vm.StructValue (≥3 fields)
// Stores hash references instead of inline field values
type StructValueRef struct {
	FieldNames  []string // Field names (parallel to FieldHashes)
	FieldHashes []Hash   // Field value hashes (parallel to FieldNames)
}

func (s *StructValueRef) Serialize(w io.Writer) error {
	// Lists are naturally deterministic, no special handling needed
	return msgpack.MarshalWrite(w, s)
}

func (s *StructValueRef) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, s)
}

// ArrayValueRef is used for large vm.ArrayValue (≥5 elements)
// Stores hash references instead of inline element values
type ArrayValueRef struct {
	ElementHashes []Hash // Hash for each element
}

func (a *ArrayValueRef) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, a)
}

func (a *ArrayValueRef) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, a)
}

// NonDetValueRef is used for vm.NonDetValue
// Stores hash references for each choice
type NonDetValueRef struct {
	ChoiceHashes []Hash // Hash for each choice
}

func (n *NonDetValueRef) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, n)
}

func (n *NonDetValueRef) Deserialize(r io.Reader) error {
	return msgpack.UnmarshalRead(r, n)
}
