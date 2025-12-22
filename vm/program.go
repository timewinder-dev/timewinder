package vm

import (
	"encoding/json"
	"errors"
	"fmt"
)

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

var ErrEndOfCode = errors.New("End of code block")

func (p *Program) GetInstruction(ptr ExecPtr) (Op, error) {
	var f *Function
	if ptr.CodeID() == 0 {
		f = p.Main
	} else {
		f = p.Code[ptr.CodeID()-1]
	}
	if len(f.Bytecode) <= ptr.Offset() {
		return Op{}, ErrEndOfCode
	}
	return f.Bytecode[ptr.Offset()], nil
}

func (p *Program) Resolve(name string) (ExecPtr, bool) {
	if v, ok := p.Definitions[name]; ok {
		return NewExecPtr(v), true
	}
	return 0, false
}

// GetFunction returns the Function for a given PC
func (p *Program) GetFunction(ptr ExecPtr) *Function {
	if ptr.CodeID() == 0 {
		return p.Main
	}
	if ptr.CodeID()-1 < len(p.Code) {
		return p.Code[ptr.CodeID()-1]
	}
	return nil
}

// GetLineNumber returns the source line number for a given PC
func (p *Program) GetLineNumber(ptr ExecPtr) int {
	var f *Function
	if ptr.CodeID() == 0 {
		f = p.Main
	} else if ptr.CodeID()-1 < len(p.Code) {
		f = p.Code[ptr.CodeID()-1]
	} else {
		return 0
	}

	offset := ptr.Offset()
	if offset >= 0 && offset < len(f.LineMap) {
		return f.LineMap[offset]
	}
	return 0
}

// GetFilename returns the source filename for a given PC
func (p *Program) GetFilename(ptr ExecPtr) string {
	var f *Function
	if ptr.CodeID() == 0 {
		f = p.Main
	} else if ptr.CodeID()-1 < len(p.Code) {
		f = p.Code[ptr.CodeID()-1]
	} else {
		return ""
	}
	return f.Filename
}

type Function struct {
	Bytecode  []Op
	Params    []FunctionParam
	LocalVars []string // UNUSED: Kept for compatibility, no longer used with new scoping rules
	LineMap   []int    // Maps PC offset to source line number
	Filename  string   // Source filename for this function
}

func (f *Function) DebugPrint() {
	fmt.Printf("Params: %#v\n", f.Params)
	for i, b := range f.Bytecode {
		fmt.Printf("  %03d: %s\n", i, b)
	}
}

type ExecPtr uint64

func (ptr ExecPtr) MarshalJSON() ([]byte, error) {
	out := make(map[string]int)
	out["offset"] = ptr.Offset()
	out["code_id"] = ptr.CodeID()
	return json.Marshal(out)
}

func (ptr ExecPtr) Offset() int {
	return int(0xFFFFFFFF & ptr)
}

func (ptr ExecPtr) CodeID() int {
	return int(ptr >> 32)
}

func (ptr ExecPtr) Inc() ExecPtr {
	return ptr + 1
}

func (ptr ExecPtr) SetOffset(off int) ExecPtr {
	return ExecPtr((ptr.CodeID() << 32) | int(0xFFFFFFFF&off))
}

func (ptr ExecPtr) String() string {
	codeID := ptr.CodeID()
	offset := ptr.Offset()
	if codeID == 0 {
		return fmt.Sprintf("main:%d", offset)
	}
	return fmt.Sprintf("fn%d:%d", codeID, offset)
}

func NewExecPtr(block int) ExecPtr {
	return ExecPtr(block << 32)
}

type FunctionParam struct {
	Name    string
	Default Value
	ArgList bool
	ArgMap  bool
}
