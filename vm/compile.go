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
	localVars   []string // Variables assigned anywhere in this function
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

// collectAssignedVars performs static analysis to find all variables
// that are assigned anywhere in the given statements (JavaScript-like scoping)
// Variables declared with global_var() are excluded (they remain global)
func collectAssignedVars(stmts []syntax.Stmt) []string {
	varsMap := make(map[string]bool)
	globalVars := make(map[string]bool)

	// First pass: collect global_var() declarations
	for _, stmt := range stmts {
		collectGlobalDeclarations(stmt, globalVars)
	}

	// Second pass: collect assignments
	for _, stmt := range stmts {
		scanForAssignments(stmt, varsMap)
	}

	// Remove global variables from the local variables list
	for varName := range globalVars {
		delete(varsMap, varName)
	}

	// Convert map to slice
	vars := make([]string, 0, len(varsMap))
	for varName := range varsMap {
		vars = append(vars, varName)
	}
	return vars
}

// scanForAssignments recursively traverses a syntax node looking for assignments
func scanForAssignments(node syntax.Node, vars map[string]bool) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *syntax.AssignStmt:
		// Handle assignments: x = 1, x += 1, x, y = 2, 3
		// LHS can be an Ident or TupleExpr (for multi-assignment)
		extractAssignedIdents(n.LHS, vars)
		// Recurse into RHS expression
		scanForAssignments(n.RHS, vars)

	case *syntax.IfStmt:
		scanForAssignments(n.Cond, vars)
		for _, stmt := range n.True {
			scanForAssignments(stmt, vars)
		}
		for _, stmt := range n.False {
			scanForAssignments(stmt, vars)
		}

	case *syntax.ForStmt:
		// For loop variables are assigned (can be Ident or TupleExpr)
		extractAssignedIdents(n.Vars, vars)
		scanForAssignments(n.X, vars)
		for _, stmt := range n.Body {
			scanForAssignments(stmt, vars)
		}

	case *syntax.WhileStmt:
		scanForAssignments(n.Cond, vars)
		for _, stmt := range n.Body {
			scanForAssignments(stmt, vars)
		}

	case *syntax.ReturnStmt:
		if n.Result != nil {
			scanForAssignments(n.Result, vars)
		}

	case *syntax.ExprStmt:
		scanForAssignments(n.X, vars)

	case *syntax.BranchStmt:
		// break, continue - no assignments

	case *syntax.DefStmt:
		// Nested function definitions - don't traverse (unsupported anyway)

	// Expression nodes (don't assign, but may contain assignments in comprehensions)
	case *syntax.BinaryExpr:
		scanForAssignments(n.X, vars)
		scanForAssignments(n.Y, vars)

	case *syntax.UnaryExpr:
		scanForAssignments(n.X, vars)

	case *syntax.ParenExpr:
		scanForAssignments(n.X, vars)

	case *syntax.CallExpr:
		scanForAssignments(n.Fn, vars)
		for _, arg := range n.Args {
			scanForAssignments(arg, vars)
		}

	case *syntax.DotExpr:
		scanForAssignments(n.X, vars)

	case *syntax.IndexExpr:
		scanForAssignments(n.X, vars)
		scanForAssignments(n.Y, vars)

	case *syntax.SliceExpr:
		scanForAssignments(n.X, vars)
		if n.Lo != nil {
			scanForAssignments(n.Lo, vars)
		}
		if n.Hi != nil {
			scanForAssignments(n.Hi, vars)
		}
		if n.Step != nil {
			scanForAssignments(n.Step, vars)
		}

	case *syntax.ListExpr:
		for _, elem := range n.List {
			scanForAssignments(elem, vars)
		}

	case *syntax.DictExpr:
		for _, entryExpr := range n.List {
			if entry, ok := entryExpr.(*syntax.DictEntry); ok {
				scanForAssignments(entry.Key, vars)
				scanForAssignments(entry.Value, vars)
			}
		}

	case *syntax.TupleExpr:
		for _, elem := range n.List {
			scanForAssignments(elem, vars)
		}

	case *syntax.Comprehension:
		// List/dict comprehensions can have assignments
		for _, clause := range n.Clauses {
			if forClause, ok := clause.(*syntax.ForClause); ok {
				if ident, ok := forClause.Vars.(*syntax.Ident); ok {
					vars[ident.Name] = true
				}
				scanForAssignments(forClause.X, vars)
			} else if ifClause, ok := clause.(*syntax.IfClause); ok {
				scanForAssignments(ifClause.Cond, vars)
			}
		}

	case *syntax.CondExpr:
		scanForAssignments(n.Cond, vars)
		scanForAssignments(n.True, vars)
		scanForAssignments(n.False, vars)

	case *syntax.LambdaExpr:
		// Lambda parameters are not assignments in outer scope
		scanForAssignments(n.Body, vars)

	// Leaf nodes (no traversal needed)
	case *syntax.Ident:
		// Just a reference, not an assignment
	case *syntax.Literal:
		// No assignments
	}
}

// collectGlobalDeclarations scans for global_var() calls and extracts variable names
func collectGlobalDeclarations(node syntax.Node, globalVars map[string]bool) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *syntax.ExprStmt:
		// Check if this is a global_var() call
		if call, ok := n.X.(*syntax.CallExpr); ok {
			if ident, ok := call.Fn.(*syntax.Ident); ok && ident.Name == "global_var" {
				// Extract variable names from arguments
				for _, arg := range call.Args {
					if lit, ok := arg.(*syntax.Literal); ok {
						// Extract string literal value
						if strVal, ok := lit.Value.(string); ok {
							globalVars[strVal] = true
						}
					}
				}
			}
		}
		// Recurse into expression
		collectGlobalDeclarations(n.X, globalVars)

	case *syntax.IfStmt:
		for _, stmt := range n.True {
			collectGlobalDeclarations(stmt, globalVars)
		}
		for _, stmt := range n.False {
			collectGlobalDeclarations(stmt, globalVars)
		}

	case *syntax.ForStmt:
		for _, stmt := range n.Body {
			collectGlobalDeclarations(stmt, globalVars)
		}

	case *syntax.WhileStmt:
		for _, stmt := range n.Body {
			collectGlobalDeclarations(stmt, globalVars)
		}

	// For other statement types, we don't need to recurse
	// (global_var() should only appear at statement level)
	}
}

// extractAssignedIdents extracts all identifiers that are being assigned to
// from an LHS expression (which can be Ident, TupleExpr, ListExpr, etc.)
func extractAssignedIdents(expr syntax.Expr, vars map[string]bool) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *syntax.Ident:
		// Simple assignment: x = 1
		vars[e.Name] = true

	case *syntax.TupleExpr:
		// Multi-assignment: x, y = 1, 2
		for _, elem := range e.List {
			extractAssignedIdents(elem, vars)
		}

	case *syntax.ListExpr:
		// List unpacking: [x, y] = [1, 2]
		for _, elem := range e.List {
			extractAssignedIdents(elem, vars)
		}

	case *syntax.ParenExpr:
		// Parenthesized: (x) = 1
		extractAssignedIdents(e.X, vars)

	// IndexExpr and DotExpr are attribute/index assignments, not variable assignments
	// e.g., obj.x = 1 or arr[0] = 1
	// These don't create new local variables
	}
}
