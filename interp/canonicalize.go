package interp

func Canonicalize(state *State) ([]*State, error) {
	queue := []*State{state}
	var done []*State
	for len(queue) != 0 {
		st := queue[0]
		queue = queue[1:]
		v, err := st.Expand()
		if err != nil {
			return nil, err
		}
		if v == nil {
			done = append(done, st)
		} else {
			for _, s := range v {
				queue = append(queue, s)
			}
		}
	}
	return done, nil
}

func (s *State) Expand() ([]*State, error) {
	v, err := s.Globals.Expand()
	if err != nil {
		return nil, err
	}
	if v != nil {
		var out []*State
		for _, x := range v {
			n := s.Clone()
			n.Globals = x
			out = append(out, n)
		}
		return out, nil
	}
	for i, l := range s.Stacks {
		for j, f := range l {
			v, err := f.Expand()
			if err != nil {
				return nil, err
			}
			if v != nil {
				var out []*State
				for _, x := range v {
					n := s.Clone()
					n.Stacks[i][j] = x
					out = append(out, n)
				}
				return out, nil
			}
		}
	}
	return nil, nil
}

func (f *StackFrame) Expand() ([]*StackFrame, error) {
	for i, v := range f.Stack {
		vals, err := v.Expand()
		if err != nil {
			return nil, err
		}
		if vals != nil {
			var out []*StackFrame
			for _, x := range vals {
				n := f.Clone()
				n.Stack[i] = x
				out = append(out, n)
			}
			return out, nil
		}
	}

	for k, v := range f.Variables {
		vals, err := v.Expand()
		if err != nil {
			return nil, err
		}
		if vals != nil {
			var out []*StackFrame
			for _, x := range vals {
				n := f.Clone()
				n.Variables[k] = x
				out = append(out, n)
			}
			return out, nil
		}
	}
	return nil, nil
}
