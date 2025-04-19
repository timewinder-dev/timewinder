package vm

import (
	"errors"

	"go.starlark.net/syntax"
)

type Op struct {
	Code Opcode
	Arg  Value
}

type compileContext struct {
	ops []Op
}

func (cc *compileContext) emit(op Opcode) {
	cc.ops = append(cc.ops, Op{Code: op, Arg: nil})
}

func (cc *compileContext) emitArg(op Opcode, val Value) {
	cc.ops = append(cc.ops, Op{Code: op, Arg: val})
}

func (cc *compileContext) genLabel() string {

}

func (cc *compileContext) newLabel(label string) {
	cc.ops = append(cc.ops)
}

func newCompileContext() *compileContext {
	return &compileContext{}
}

func Compile(file *syntax.File) (*Program, error) {
	p := &Program{
		Definitions: make(map[string]*Function),
	}
	// Top level context
	env := newCompileContext()
	for _, s := range file.Stmts {
		err := env.statement(s)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (cc *compileContext) statement(s syntax.Stmt) error {
	switch v := s.(type) {
	case *syntax.AssignStmt:
	case *syntax.BranchStmt:
	case *syntax.DefStmt:
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
	case *syntax.WhileStmt:
	case *syntax.IfStmt:
	case *syntax.LoadStmt:
		return errors.New("LoadStmt is still unhandled")
	case *syntax.ReturnStmt:
	default:
		panic("Unhandled statment type")
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
	case *syntax.CallExpr:
	case *syntax.Comprehension:
	case *syntax.CondExpr:
	case *syntax.DictEntry:
	case *syntax.DictExpr:
	case *syntax.DotExpr:
	case *syntax.Ident:
	case *syntax.IndexExpr:
	case *syntax.LambdaExpr:
		return errors.New("Lambda expressions are unsupported")
	case *syntax.ListExpr:
	case *syntax.Literal:
	case *syntax.ParenExpr:
	case *syntax.SliceExpr:
	case *syntax.TupleExpr:
	case *syntax.UnaryExpr:
	default:
		panic("Unhandled expr type")
	}
	return nil
}
