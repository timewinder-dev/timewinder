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
	ops         []Op
	topLevel    bool
	subContext  map[string]*compileContext
	params      []FunctionParam
	localVars   []string // UNUSED: Kept for compatibility, no longer tracked with new scoping rules
	lineMap     []int    // Maps op index to source line number
	filename    string   // Source filename
	currentLine int      // Current source line being compiled
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
	// Record the source line for this op
	cc.lineMap = append(cc.lineMap, cc.currentLine)
}

func (cc *compileContext) newLabel() string {
	return uuid.NewString()
}

func (cc *compileContext) emitLabel(s string) {
	cc.ops = append(cc.ops, Op{Code: LABEL, Arg: StrValue(s)})
	cc.lineMap = append(cc.lineMap, cc.currentLine)
}

// setLine sets the current source line from a syntax node
func (cc *compileContext) setLine(node syntax.Node) {
	if node != nil {
		start, _ := node.Span()
		cc.currentLine = int(start.Line)
	}
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
	prog, err := Compile(synFile)
	if err != nil {
		return nil, err
	}
	// Set filename for main and all functions
	prog.Main.Filename = path
	for _, fn := range prog.Code {
		fn.Filename = path
	}
	return prog, nil
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
	f.LocalVars = cc.localVars
	offsetmap := make(map[string]int)
	// Build bytecode and lineMap, skipping LABEL ops
	for i, b := range cc.ops {
		if b.Code == LABEL {
			offsetmap[string(b.Arg.(StrValue))] = len(f.Bytecode)
			continue
		}
		f.Bytecode = append(f.Bytecode, b)
		// Copy line number for this op (accounting for skipped labels)
		if i < len(cc.lineMap) {
			f.LineMap = append(f.LineMap, cc.lineMap[i])
		} else {
			f.LineMap = append(f.LineMap, 0)
		}
	}
	// Resolve label references to actual offsets
	for i, b := range f.Bytecode {
		switch b.Code {
		case JMP:
			fallthrough
		case ITER_START:
			fallthrough
		case ITER_START_2:
			fallthrough
		case JFALSE:
			fallthrough
		case CONDITIONAL_YIELD:
			fallthrough
		case CONDITIONAL_FAIR_YIELD:
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

// NOTE: The following functions have been removed as they're no longer needed
// with the new unified namespace scoping rules:
// - collectAssignedVars: Previously tracked which variables were assigned in a function
// - scanForAssignments: Helper for collectAssignedVars
// - collectGlobalDeclarations: Previously detected global_var() calls
// - extractAssignedIdents: Helper for scanForAssignments
//
// The new scoping rules are simpler:
// - Variables in global scope are automatically accessible/writable from functions
// - Variables not in global scope create new locals on first write
// - Shadowing (same name in both scopes) produces an error
// See interp/step.go SETVAL and resolveVar for implementation
