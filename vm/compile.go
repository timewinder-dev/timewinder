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
		n := len(p.Code)
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

func (cc *compileContext) statement(s syntax.Stmt) error {
	switch v := s.(type) {
	case *syntax.AssignStmt:
		return cc.assign(v.Op, v.LHS, v.RHS)
	//case *syntax.BranchStmt:
	case *syntax.DefStmt:
		if !cc.topLevel {
			return errors.New("Nested defs are unsupported")
		}
		sub := newCompileContext()
		name := v.Name.Name
		var err error
		sub.params, err = getFunctionParams(v.Params)
		if err != nil {
			return err
		}
		err = sub.buildFromStatements(v.Body)
		if err != nil {
			return err
		}
		cc.subContext[name] = sub
	case *syntax.ExprStmt:
		if _, ok := v.X.(*syntax.Literal); ok {
			// Opt: don't compile literals only to pop them.
			return nil
		}
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		cc.emit(POP)
	case *syntax.ForStmt:
		idents := 0
		switch vars := v.Vars.(type) {
		case *syntax.Ident:
			cc.emit(PUSH, StrValue(vars.Name))
			idents = 1
		case *syntax.TupleExpr:
			if len(vars.List) > 2 {
				return errors.New("Too many variables in for list")
			}
			idents = len(vars.List)
			for _, id := range vars.List {
				if v, ok := id.(*syntax.Ident); ok {
					cc.emit(PUSH, StrValue(v.Name))
				} else {
					return errors.New("Non-identifier in for variable")
				}
			}
		default:
			return errors.New("Unsupported for variables")
		}
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		endLabel := cc.newLabel()
		if idents == 1 {
			cc.emit(ITER_START, StrValue(endLabel))
		} else if idents == 2 {
			cc.emit(ITER_START_2, StrValue(endLabel))
		} else {
			return errors.New("Too many identifiers")
		}
		err = cc.buildFromStatements(v.Body)
		if err != nil {
			return err
		}
		cc.emit(ITER_NEXT)
		cc.emit(LABEL, StrValue(endLabel))
	//case *syntax.WhileStmt:
	case *syntax.IfStmt:
		err := cc.expr(v.Cond)
		if err != nil {
			return err
		}
		label := cc.newLabel()
		cc.emit(JFALSE, StrValue(label))
		cc.buildFromStatements(v.True)
		if len(v.False) == 0 {
			cc.emitLabel(label)
			return nil
		}
		endLabel := cc.newLabel()
		cc.emit(JMP, StrValue(endLabel))
		cc.emitLabel(label)
		cc.buildFromStatements(v.False)
		cc.emitLabel(endLabel)
	case *syntax.LoadStmt:
		return errors.New("LoadStmt is unimplemented")
	case *syntax.ReturnStmt:
		if v.Result == nil {
			cc.emit(PUSH, None)
		} else {
			err := cc.expr(v.Result)
			if err != nil {
				return err
			}
		}
		cc.emit(RETURN)
	default:
		return fmt.Errorf("Unhandled statment type %T", s)
	}
	return nil
}

func (cc *compileContext) expr(e syntax.Expr) error {
	switch v := e.(type) {
	case *syntax.BinaryExpr:
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		err = cc.expr(v.Y)
		if err != nil {
			return err
		}
		return cc.binOp(v.Op)
	case *syntax.CallExpr:
		if ok, err := cc.specialCall(v); ok {
			return err
		}
		for _, a := range v.Args {
			err := cc.callArg(a)
			if err != nil {
				return err
			}
		}
		err := cc.expr(v.Fn)
		if err != nil {
			return err
		}
		cc.emit(CALL, IntValue(len(v.Args)))
	case *syntax.Comprehension:
		return errors.New("Comprehensions are as yet unsupported")
	case *syntax.CondExpr:
		err := cc.expr(v.Cond)
		if err != nil {
			return err
		}
		label := cc.newLabel()
		cc.emit(JFALSE, StrValue(label))
		err = cc.expr(v.True)
		if err != nil {
			return err
		}
		endLabel := cc.newLabel()
		cc.emit(JMP, StrValue(endLabel))
		cc.emitLabel(label)
		err = cc.expr(v.False)
		if err != nil {
			return err
		}
		cc.emitLabel(endLabel)
	case *syntax.DictEntry:
		err := cc.expr(v.Key)
		if err != nil {
			return err
		}
		err = cc.expr(v.Value)
		if err != nil {
			return err
		}
		cc.emit(BUILD_LIST, IntValue(2))
	case *syntax.DictExpr:
		for _, expr := range v.List {
			err := cc.expr(expr)
			if err != nil {
				return err
			}
		}
		cc.emit(BUILD_DICT, IntValue(len(v.List)))
	case *syntax.DotExpr:
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		cc.emit(PUSH, StrValue(v.Name.Name))
		cc.emit(GETATTR)
	case *syntax.Ident:
		if v.Name == "True" {
			cc.emit(PUSH, BoolTrue)
			return nil
		}
		if v.Name == "False" {
			cc.emit(PUSH, BoolFalse)
			return nil
		}
		cc.emit(PUSH, StrValue(v.Name))
		cc.emit(GETVAL)
	case *syntax.IndexExpr:
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		err = cc.expr(v.Y)
		if err != nil {
			return err
		}
		cc.emit(GETATTR)
	case *syntax.LambdaExpr:
		return errors.New("Lambda expressions are unsupported")
	case *syntax.ListExpr:
		for _, exp := range v.List {
			err := cc.expr(exp)
			if err != nil {
				return err
			}
		}
		cc.emit(BUILD_LIST, IntValue(len(v.List)))
	case *syntax.Literal:
		val, err := litToValue(v.Value)
		if err != nil {
			return err
		}
		cc.emit(PUSH, val)
	//case *syntax.ParenExpr:
	//case *syntax.SliceExpr:
	case *syntax.TupleExpr:
		for _, exp := range v.List {
			err := cc.expr(exp)
			if err != nil {
				return err
			}
		}
		cc.emit(BUILD_LIST, IntValue(len(v.List)))
	case *syntax.UnaryExpr:
		return cc.unary(v)
	default:
		return fmt.Errorf("Unhandled expr type %T", e)
	}
	return nil
}

func (cc *compileContext) binOp(op syntax.Token) error {
	switch op {
	case syntax.PLUS: // +
		cc.emit(ADD)
	case syntax.MINUS: // -
		cc.emit(SUBTRACT)
	case syntax.STAR: // *
		cc.emit(MULTIPLY)
	case syntax.SLASH: // /
		cc.emit(DIVIDE)
	//case syntax.SLASHSLASH: // //
	//case syntax.PERCENT: // %
	//case syntax.AMP: // &
	//case syntax.PIPE: // |
	//case syntax.CIRCUMFLEX: // ^
	//case syntax.LTLT: // <<
	//case syntax.GTGT: // >>
	//case syntax.TILDE: // ~
	//case syntax.DOT: // .
	//case syntax.COMMA: // ,
	//case syntax.EQ: // =
	//case syntax.SEMI: // ;
	//case syntax.COLON: // :
	//case syntax.LPAREN: // (
	//case syntax.RPAREN: // )
	//case syntax.LBRACK: // [
	//case syntax.RBRACK: // ]
	//case syntax.LBRACE: // {
	//case syntax.RBRACE: // }
	case syntax.LT: // <
		cc.emit(LT)
	case syntax.GT: // >
		cc.emit(LTE)
		cc.emit(NOT)
	case syntax.GE: // >=
		cc.emit(LT)
		cc.emit(NOT)
	case syntax.LE: // <=
		cc.emit(LTE)
	case syntax.EQL: // ==
		cc.emit(EQ)
	case syntax.NEQ: // !=
		cc.emit(EQ)
		cc.emit(NOT)
	//case syntax.PLUS_EQ: // +=    (keep order consistent with PLUS..GTGT)
	//case syntax.MINUS_EQ: // -=
	//case syntax.STAR_EQ: // *=
	//case syntax.SLASH_EQ: // /=
	//case syntax.SLASHSLASH_EQ: // //=
	//case syntax.PERCENT_EQ: // %=
	//case syntax.AMP_EQ: // &=
	//case syntax.PIPE_EQ: // |=
	//case syntax.CIRCUMFLEX_EQ: // ^=
	//case syntax.LTLT_EQ: // <<=
	//case syntax.GTGT_EQ: // >>=
	//case syntax.STARSTAR: // **

	//// Keywords
	//case syntax.AND:
	//case syntax.BREAK:
	//case syntax.CONTINUE:
	//case syntax.DEF:
	//case syntax.ELIF:
	//case syntax.ELSE:
	//case syntax.FOR:
	//case syntax.IF:
	//case syntax.IN:
	//case syntax.LAMBDA:
	//case syntax.LOAD:
	//case syntax.NOT:
	//case syntax.NOT_IN: // synthesized by parser from NOT IN
	//case syntax.OR:
	//case syntax.PASS:
	//case syntax.RETURN:
	//case syntax.WHILE:
	default:
		return fmt.Errorf("compileContext: Unhandled binary operation %#v", op)
	}
	return nil
}

func (cc *compileContext) unary(e *syntax.UnaryExpr) error {
	switch e.Op {
	default:
		return fmt.Errorf("compileContext: Unhandled unary operation %#v", e.Op.String())
	}
}

func (cc *compileContext) callArg(arg syntax.Expr) error {
	fmt.Printf("callArg: %T %#v\n", arg, arg)
	return nil
}

func (cc *compileContext) specialCall(call *syntax.CallExpr) (bool, error) {
	return false, nil
}

func (cc *compileContext) assign(op syntax.Token, lhs syntax.Expr, rhs syntax.Expr) error {
	err := cc.expr(rhs)
	if err != nil {
		return err
	}
	if op != syntax.EQ {
		err := cc.assignSelfReassign(op, lhs)
		if err != nil {
			return err
		}
	}
	switch v := lhs.(type) {
	case *syntax.Ident:
		if v.Name == "True" || v.Name == "False" {
			return fmt.Errorf("Reassigning `%s` is not allowed", v.Name)
		}
		cc.emit(PUSH, StrValue(v.Name))
		cc.emit(SETVAL)
	case *syntax.IndexExpr:
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		err = cc.expr(v.Y)
		if err != nil {
			return err
		}
		cc.emit(SETATTR)
	case *syntax.DotExpr:
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		cc.emit(PUSH, StrValue(v.Name.Name))
		cc.emit(SETATTR)
	default:
		return fmt.Errorf("assign: Unhandled LHS expr type %T", lhs)
	}
	return nil
}

func (cc *compileContext) assignSelfReassign(op syntax.Token, lhs syntax.Expr) error {
	err := cc.expr(lhs)
	if err != nil {
		return err
	}
	switch op {
	case syntax.PLUS_EQ:
		cc.emit(ADD)
	case syntax.MINUS_EQ:
		cc.emit(SWAP)
		cc.emit(SUBTRACT)
	default:
		return fmt.Errorf("%#v assignments unimplemented", op)
	}
	return nil
}

func getFunctionParams(e []syntax.Expr) ([]FunctionParam, error) {
	var out []FunctionParam
	for _, x := range e {
		switch v := x.(type) {
		case *syntax.Ident:
			out = append(out, FunctionParam{Name: v.Name})
		default:
			return nil, fmt.Errorf("Unhandled function param expr type %T", x)
		}
	}
	return out, nil
}

func unparen(e syntax.Expr) syntax.Expr {
	if p, ok := e.(*syntax.ParenExpr); ok {
		return unparen(p.X)
	}
	return e
}

func litToValue(l any) (Value, error) {
	switch t := l.(type) {
	case int64:
		return IntValue(int(t)), nil
	case string:
		return StrValue(t), nil
	case float64:
		return FloatValue(t), nil
	}
	return nil, fmt.Errorf("litToValue: Unsupported literal value type %T", l)
}
