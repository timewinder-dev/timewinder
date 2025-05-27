package interp

import (
	"errors"
	"io"

	"github.com/shamaton/msgpack/v2"
	"github.com/timewinder-dev/timewinder/vm"
)

func NewState() *State {
	return &State{
		Globals: &StackFrame{},
	}
}

func (s *State) Clone() *State {
	out := &State{
		Globals: s.Globals.Clone(),
	}
	for _, stack := range s.Stacks {
		var new []*StackFrame
		for _, x := range stack {
			new = append(new, x.Clone())
		}
		out.Stacks = append(out.Stacks, new)
	}
	return out
}

func (s *State) Serialize(w io.Writer) error {
	return msgpack.MarshalWrite(w, s)
}

func (s *State) Deserialize(r io.Reader) error {
	return errors.New("deserialize unimplemented")
}

func (s *State) AddThread(frame *StackFrame) {
	s.Stacks = append(s.Stacks, []*StackFrame{frame})
}

func (f *StackFrame) Pop() vm.Value {
	if len(f.Stack) == 0 {
		panic("Stack underrun")
		//return vm.None
	}
	v := f.Stack[len(f.Stack)-1]
	f.Stack = f.Stack[:len(f.Stack)-1]
	return v
}

func (f *StackFrame) Push(v vm.Value) {
	f.Stack = append(f.Stack, v)
}

func (f *StackFrame) Clone() *StackFrame {
	out := &StackFrame{
		PC:          f.PC,
		PauseReason: f.PauseReason,
	}
	for _, v := range f.Stack {
		out.Stack = append(out.Stack, v.Clone())
	}
	for k, v := range f.Variables {
		out.StoreVar(k, v.Clone())
	}
	for _, i := range f.IteratorStack {
		out.IteratorStack = append(out.IteratorStack, i.Clone())
	}
	return out
}

func (f *StackFrame) StoreVar(key string, value vm.Value) {
	if f.Variables == nil {
		f.Variables = make(map[string]vm.Value)
	}
	f.Variables[key] = value
}

func (f *StackFrame) Has(key string) bool {
	if f.Variables == nil {
		return false
	}
	_, ok := f.Variables[key]
	return ok
}
