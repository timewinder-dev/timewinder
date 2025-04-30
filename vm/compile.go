package vm

import (
	"errors"
	"fmt"

	"go.starlark.net/syntax"
)

type Op struct {
	Code Opcode
	Arg  Value
}

type compileContext struct {
	ops        []Op
	topLevel   bool
	subContext map[string]*compileContext
}

func (cc *compileContext) emit(op Opcode) {
	cc.ops = append(cc.ops, Op{Code: op, Arg: nil})
}

func (cc *compileContext) emitArg(op Opcode, val Value) {
	cc.ops = append(cc.ops, Op{Code: op, Arg: val})
}

func (cc *compileContext) newLabel(label string) {
	cc.ops = append(cc.ops, Op{Code: NOP, Arg: StrValue(label)})
}

func newCompileContext() *compileContext {
	return &compileContext{
		subContext: make(map[string]*compileContext),
	}
}

func Compile(file *syntax.File) (*Program, error) {
	p := &Program{
		Definitions: make(map[string]int),
		Predicates:  make(map[string]int),
	}
	// Top level context
	return p, nil
}

func buildCompileContextTree(file *syntax.File) (*compileContext, error) {
	cc := newCompileContext()
	cc.topLevel = true
	for _, s := range file.Stmts {
		err := cc.statement(s)
		if err != nil {
			return nil, err
		}
	}
	return cc, nil
}

func (cc *compileContext) statement(s syntax.Stmt) error {
	switch v := s.(type) {
	//case *syntax.AssignStmt:
	//case *syntax.BranchStmt:
	//case *syntax.DefStmt:
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
	//case *syntax.ForStmt:
	//case *syntax.WhileStmt:
	//case *syntax.IfStmt:
	case *syntax.LoadStmt:
		return errors.New("LoadStmt is unimplemented")
	//case *syntax.ReturnStmt:
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
	//case *syntax.CallExpr:
	//case *syntax.Comprehension:
	//case *syntax.CondExpr:
	//case *syntax.DictEntry:
	//case *syntax.DictExpr:
	//case *syntax.DotExpr:
	//case *syntax.Ident:
	//case *syntax.IndexExpr:
	case *syntax.LambdaExpr:
		return errors.New("Lambda expressions are unsupported")
		//case *syntax.ListExpr:
		//case *syntax.Literal:
		//case *syntax.ParenExpr:
		//case *syntax.SliceExpr:
		//case *syntax.TupleExpr:
		//case *syntax.UnaryExpr:
	default:
		return fmt.Errorf("Unhandled expr type %T", e)
	}
	return nil
}

func (cc *compileContext) binOp(op syntax.Token) error {
	switch op {
	//case syntax.PLUS: // +
	//case syntax.MINUS: // -
	//case syntax.STAR: // *
	//case syntax.SLASH: // /
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
	//case syntax.LT: // <
	//case syntax.GT: // >
	//case syntax.GE: // >=
	//case syntax.LE: // <=
	//case syntax.EQL: // ==
	//case syntax.NEQ: // !=
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
