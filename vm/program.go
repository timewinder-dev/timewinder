package vm

import "fmt"

type Program struct {
	Definitions map[string]int
	Code        []*Function
	Main        *Function
}

func (p *Program) DebugPrint() {
	fmt.Printf("Defs: %#v\n", p.Definitions)
	fmt.Println("*** Main")
	p.Main.DebugPrint()
	for i, f := range p.Code {
		fmt.Printf("*** %d:\n", i)
		f.DebugPrint()
	}
}

type Function struct {
	Bytecode []Op
	Params   []FunctionParam
}

func (f *Function) DebugPrint() {
	fmt.Printf("Params: %#v\n", f.Params)
	for i, b := range f.Bytecode {
		fmt.Printf("  %03d: %s\n", i, b)
	}
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
