package vm

type Program struct {
	Definitions map[string]*Function
}

type Function struct {
	Bytecode []Opcode
}
