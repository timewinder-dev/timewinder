package vm

type Program struct {
	Definitions map[string]int
	Predicates  map[string]int
	Code        []*Function
}

type Function struct {
	Bytecode []Opcode
}

type ExecPtr uint64

func (ptr ExecPtr) Offset() int {
	return int(0xFFFFFFFF & ptr)
}

func (ptr ExecPtr) CodeID() int {
	return int(ptr >> 32)
}

type FunctionParam struct {
	Name    string
	Default Value
	ArgList bool
	ArgMap  bool
}
