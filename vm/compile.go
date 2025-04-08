package vm

import "go.starlark.net/syntax"

type compileContext struct {
}

func newCompileContext() *compileContext {
	return &compileContext{}
}

func Compile(file *syntax.File) (*Program, error) {
	p := &Program{
		Definitions: make(map[string]*Function),
	}
	for _, s := range file.Stmts {
		env := newCompileContext()
		env.statement(s)
	}
	return p, nil
}

func (cc *compileContext) statement(s syntax.Stmt) {

}
