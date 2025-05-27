package model

type SingleThreadEngine struct {
	Queue     []Thunk
	NextQueue []Thunk
	Executor  *Executor
}

func InitSingleThread(exec *Executor) (*SingleThreadEngine, error) {
	allStates, err := interp.Canonicalize(exec.InitialState)
	if err != nil {
		return nil, err
	}
	st := &SingleThreadEngine{
		Executor: exec,
	}
	for _, s := range allStates {
		for i := 0; i < len(exec.Threads); i++ {
			st.Queue = append(st.Queue, Thunk{
				ToRun: i,
				State: s,
			})
		}
	}
	return st, nil
}

func (s *SingleThreadEngine) RunModel() error {

}
