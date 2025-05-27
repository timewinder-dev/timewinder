package vm

import (
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"go.starlark.net/syntax"
)

type Op struct {
	Code Opcode
	Arg  Value
}

func (o Op) String() string {
	if o.Arg == nil {
		return o.Code.String()
	}
	return fmt.Sprintf("%s %v", o.Code, o.Arg)
}

type compileContext struct {
	ops        []Op
	topLevel   bool
	subContext map[string]*compileContext
	params     []FunctionParam
}

func (cc *compileContext) DebugPrint() {
	fmt.Printf("ops: %#v\n", cc.ops)
	fmt.Printf("params: %#v\n", cc.params)
	if len(cc.subContext) != 0 {
		for k, v := range cc.subContext {
			fmt.Printf("%s:\n", k)
			fmt.Printf("\tops: %#v\n", v.ops)
			fmt.Printf("\tparams: %#v\n", v.params)
		}
	}
}

func (cc *compileContext) emit(op Opcode, val ...Value) {
	if len(val) == 0 {
		cc.ops = append(cc.ops, Op{Code: op, Arg: nil})
	} else if len(val) == 1 {
		cc.ops = append(cc.ops, Op{Code: op, Arg: val[0]})
	} else {
		panic("more than one arg to an op")
	}
}

func (cc *compileContext) newLabel() string {
	return uuid.NewString()
}

func (cc *compileContext) emitLabel(s string) {
	cc.ops = append(cc.ops, Op{Code: LABEL, Arg: StrValue(s)})
}

func newCompileContext() *compileContext {
	return &compileContext{
		subContext: make(map[string]*compileContext),
	}
}

func CompileLiteral(code string) (*Program, error) {
	opts := syntax.FileOptions{}
	synFile, err := opts.Parse("literal", code, 0)
	if err != nil {
		return nil, err
	}
	return Compile(synFile)
}

func CompilePath(path string) (*Program, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opts := syntax.FileOptions{}
	synFile, err := opts.Parse(path, f, 0)
	if err != nil {
		return nil, err
	}
	return Compile(synFile)
}

func Compile(file *syntax.File) (*Program, error) {
	cc, err := buildCompileContextTree(file)
	if err != nil {
		return nil, err
	}
	return cc.intoProgram()
}

func CompileExpr(expr string) (*Program, error) {
	opts := syntax.FileOptions{}
	exp, err := opts.ParseExpr("immediate", expr, 0)
	if err != nil {
		return nil, err
	}
	cc := newCompileContext()
	cc.topLevel = true
	err = cc.expr(exp)
	if err != nil {
		return nil, err
	}
	return cc.intoProgram()
}

func (cc *compileContext) intoProgram() (*Program, error) {
	p := &Program{
		Definitions: make(map[string]int),
	}
	if !cc.topLevel {
		return nil, errors.New("Can't make a program out of a non-top-level context")
	}
	f, err := cc.intoFunction()
	if err != nil {
		return nil, err
	}
	p.Main = f
	for k, v := range cc.subContext {
		f, err := v.intoFunction()
		if err != nil {
			return nil, err
		}
		n := len(p.Code) + 1
		p.Code = append(p.Code, f)
		p.Definitions[k] = n
	}
	// Top level context
	return p, nil
}

func (cc *compileContext) intoFunction() (*Function, error) {
	f := &Function{}
	f.Params = cc.params
	offsetmap := make(map[string]int)
	for _, b := range cc.ops {
		if b.Code == LABEL {
			offsetmap[string(b.Arg.(StrValue))] = len(f.Bytecode)
			continue
		}
		f.Bytecode = append(f.Bytecode, b)
	}
	for i, b := range f.Bytecode {
		switch b.Code {
		case JMP:
			fallthrough
		case ITER_START:
			fallthrough
		case JFALSE:
			if v, ok := b.Arg.(StrValue); ok {
				b.Arg = IntValue(offsetmap[string(v)])
			}
		}
		f.Bytecode[i] = b // Replace after changes
	}
	return f, nil
}

func buildCompileContextTree(file *syntax.File) (*compileContext, error) {
	cc := newCompileContext()
	cc.topLevel = true
	err := cc.buildFromStatements(file.Stmts)
	if err != nil {
		return nil, err
	}
	return cc, nil
}

func (cc *compileContext) buildFromStatements(stmts []syntax.Stmt) error {
	for _, s := range stmts {
		err := cc.statement(s)
		if err != nil {
			return err
		}
	}
	return nil
}
