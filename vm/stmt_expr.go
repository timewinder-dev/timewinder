package vm

import (
	"errors"
	"fmt"

	"go.starlark.net/syntax"
)

func (cc *compileContext) statement(s syntax.Stmt) error {
	// Record source line for this statement
	cc.setLine(s)

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
		// Static analysis: collect all assigned variables for JavaScript-like scoping
		sub.localVars = collectAssignedVars(v.Body)
		err = sub.buildFromStatements(v.Body)
		if err != nil {
			return err
		}
		// Add implicit return at end of function if not already present
		if len(sub.ops) == 0 || sub.ops[len(sub.ops)-1].Code != RETURN {
			sub.emit(PUSH, None)
			sub.emit(RETURN)
		}
		cc.subContext[name] = sub
	case *syntax.ExprStmt:
		// Check if this is a method call on a simple variable (e.g., queue.append(val))
		// For these, we need to store the result back to the receiver variable
		if callExpr, ok := v.X.(*syntax.CallExpr); ok {
			if dotExpr, ok := callExpr.Fn.(*syntax.DotExpr); ok {
				if ident, ok := dotExpr.X.(*syntax.Ident); ok {
					// This is varname.method(args) - emit method call then store result back
					err := cc.expr(v.X) // Emits CALL_METHOD, pushes result to stack
					if err != nil {
						return err
					}
					// Store result back to the receiver variable
					cc.emit(PUSH, StrValue(ident.Name))
					cc.emit(SETVAL)
					return nil
				}
			}
		}

		// Not a method call on simple variable, handle normally
		if _, ok := v.X.(*syntax.Literal); ok {
			// Opt: don't compile literals only to pop them.
			return nil
		}
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		// All expressions leave a value on the stack, so always POP it
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
	case *syntax.WhileStmt:
		// while condition:
		//   body
		// Compiles to:
		//   start_label:
		//     <condition>
		//     JFALSE end_label  ; JFALSE consumes the condition value
		//     <body>
		//     JMP start_label
		//   end_label:
		startLabel := cc.newLabel()
		endLabel := cc.newLabel()
		cc.emitLabel(startLabel)
		err := cc.expr(v.Cond)
		if err != nil {
			return err
		}
		cc.emit(JFALSE, StrValue(endLabel))
		// No POP needed - JFALSE already consumed the condition
		err = cc.buildFromStatements(v.Body)
		if err != nil {
			return err
		}
		cc.emit(JMP, StrValue(startLabel))
		cc.emitLabel(endLabel)
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
	// Record source line for this expression
	cc.setLine(e)

	switch v := e.(type) {
	case *syntax.BinaryExpr:
		// Handle short-circuit operators (AND, OR) specially
		if v.Op == syntax.AND || v.Op == syntax.OR {
			return cc.shortCircuitBinOp(v)
		}
		// Regular binary operators - evaluate both sides first
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

		// Check if this is a method call: obj.method(args)
		if dotExpr, ok := v.Fn.(*syntax.DotExpr); ok {
			// This is a method call
			// Stack layout: arg1, arg2, ..., argN, receiver, methodName, N
			for _, a := range v.Args {
				err := cc.callArg(a)
				if err != nil {
					return err
				}
			}
			// Push receiver
			err := cc.expr(dotExpr.X)
			if err != nil {
				return err
			}
			// Push method name
			cc.emit(PUSH, StrValue(dotExpr.Name.Name))
			// Emit CALL_METHOD with argument count
			cc.emit(CALL_METHOD, IntValue(len(v.Args)))
		} else {
			// Regular function call
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
		}
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
		if v.Name == "None" {
			cc.emit(PUSH, None)
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
	case *syntax.ParenExpr:
		return cc.expr(unparen(v))
	case *syntax.SliceExpr:
		// array[start:end:step] - step is not supported yet
		if v.Step != nil {
			return errors.New("Slice step is not supported")
		}
		// Push the array being sliced
		err := cc.expr(v.X)
		if err != nil {
			return err
		}
		// Push start index (or None if omitted)
		if v.Lo != nil {
			err = cc.expr(v.Lo)
			if err != nil {
				return err
			}
		} else {
			cc.emit(PUSH, None)
		}
		// Push end index (or None if omitted)
		if v.Hi != nil {
			err = cc.expr(v.Hi)
			if err != nil {
				return err
			}
		} else {
			cc.emit(PUSH, None)
		}
		cc.emit(SLICE)
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

// shortCircuitBinOp handles AND and OR operators with short-circuit evaluation
func (cc *compileContext) shortCircuitBinOp(e *syntax.BinaryExpr) error {
	if e.Op == syntax.AND {
		// AND short-circuit: if left is false, skip right and return false
		// Code pattern:
		//   eval left
		//   DUP               ; duplicate for testing
		//   JFALSE end_label  ; if false, skip right side
		//   POP               ; remove the duplicate (left was truthy)
		//   eval right
		//   end_label:
		//   ; result is on stack (left if it was false, right otherwise)

		err := cc.expr(e.X)
		if err != nil {
			return err
		}
		endLabel := cc.newLabel()
		cc.emit(DUP)
		cc.emit(JFALSE, StrValue(endLabel))
		cc.emit(POP) // Remove the duplicate left value (which was truthy)
		err = cc.expr(e.Y)
		if err != nil {
			return err
		}
		cc.emitLabel(endLabel)
		return nil
	}

	if e.Op == syntax.OR {
		// OR short-circuit: if left is true, skip right and return true
		// Code pattern:
		//   eval left
		//   DUP               ; duplicate for testing
		//   JFALSE else_label ; if false, try right side
		//   JMP end_label     ; if true, skip right side (keeping duplicate)
		//   else_label:
		//   POP               ; remove the duplicate false value
		//   eval right
		//   end_label:
		//   ; result is on stack (left if it was true, right otherwise)

		err := cc.expr(e.X)
		if err != nil {
			return err
		}
		elseLabel := cc.newLabel()
		endLabel := cc.newLabel()
		cc.emit(DUP)
		cc.emit(JFALSE, StrValue(elseLabel))
		// Left was truthy, skip right side
		cc.emit(JMP, StrValue(endLabel))
		cc.emitLabel(elseLabel)
		// Left was falsy, eval right side
		cc.emit(POP) // Remove the duplicate false value
		err = cc.expr(e.Y)
		if err != nil {
			return err
		}
		cc.emitLabel(endLabel)
		return nil
	}

	return fmt.Errorf("shortCircuitBinOp: unexpected op %v", e.Op)
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
	err := cc.expr(e.X)
	if err != nil {
		return err
	}
	switch e.Op {
	case syntax.NOT:
		cc.emit(NOT)
	case syntax.MINUS:
		// Unary minus: 0 - x
		cc.emit(PUSH, IntValue(0))
		cc.emit(SWAP)
		cc.emit(SUBTRACT)
	case syntax.PLUS:
		// Unary plus is a no-op
		// Value is already on stack
	default:
		return fmt.Errorf("compileContext: Unhandled unary operation %#v", e.Op.String())
	}
	return nil
}

func (cc *compileContext) callArg(arg syntax.Expr) error {
	switch v := arg.(type) {
	case *syntax.BinaryExpr:
		if v.Op == syntax.EQ {
			// Keyword argument: name=value
			if g, ok := v.X.(*syntax.Ident); ok {
				err := cc.expr(v.Y)
				if err != nil {
					return err
				}
				cc.emit(PUSH, StrValue(g.Name))
				cc.emit(BUILD_ARG)
			} else {
				return fmt.Errorf("Only identifiers are allowed on the left-hand side of a function call argument")
			}
			return nil
		}
		// For other binary expressions (like subtraction), fall through to regular expression handling
	case *syntax.UnaryExpr:
		if v.Op == syntax.STAR || v.Op == syntax.STARSTAR {
			return fmt.Errorf("Splats are currently unsupported")
		}
	}
	// fallthrough
	err := cc.expr(arg)
	if err != nil {
		return err
	}
	cc.emit(PUSH, None)
	cc.emit(BUILD_ARG)

	return nil
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
		case *syntax.BinaryExpr:
			if v.Op != syntax.EQ {
				return nil, fmt.Errorf("Only assignments are allowed within a function parameter")
			}
			if arg, ok := v.X.(*syntax.Ident); ok {
				switch y := v.Y.(type) {
				case *syntax.Literal:
					val, err := litToValue(y.Value)
					if err != nil {
						return nil, err
					}
					out = append(out, FunctionParam{Name: arg.Name, Default: val})
				default:
					return nil, fmt.Errorf("Only literals are supported as default arguments to functions")
				}

			}
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
