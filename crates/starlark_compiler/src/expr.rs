use std::collections::HashMap;

use anyhow::Result;
use starlark_syntax::syntax::ast::{AssignP, AssignTargetP, AstExprP, AstPayload};
use starlark_syntax::syntax::ast::{AstStmt, ExprP, StmtP};
use vm::bytecode::Instruction;
use vm::bytecode::TempInstruction;
use vm::Block;

pub fn compile_stmt(
    stmt: &AstStmt,
    into_block: &mut Block,
    file: &mut vm::BytecodeFile,
) -> Result<()> {
    let cur_span: vm::Span = (&stmt.span).into();
    match &stmt.node {
        StmtP::Break => {
            into_block.add_instruction(Instruction::TempInst(TempInstruction::Break), cur_span)
        }
        StmtP::Continue => {
            into_block.add_instruction(Instruction::TempInst(TempInstruction::Continue), cur_span)
        }
        StmtP::Pass => into_block.add_instruction(Instruction::NoOp, cur_span),
        StmtP::Return(expr) => {
            match expr {
                Some(ref e) => {
                    compile_expr(e, into_block, file)?;
                }
                None => into_block.add_instruction(
                    Instruction::PushLiteral(vm::value::Val::Null),
                    cur_span.clone(),
                ),
            };
            into_block.add_instruction(Instruction::Return, cur_span);
        }
        StmtP::Expression(expr) => compile_expr(expr, into_block, file)?,
        StmtP::Assign(assign) => compile_assign(assign, into_block, file)?,
        StmtP::AssignModify(_, _, _) => todo!(),
        StmtP::Statements(stmts) => {
            for s in stmts {
                compile_stmt(s, into_block, file)?
            }
        }
        StmtP::If(if_expr, body_stmt) => {
            let mut body = Block::default();
            compile_stmt(body_stmt, &mut body, file)?;
            compile_expr(if_expr, into_block, file)?;
            into_block.add_instruction(
                Instruction::RelJumpIfFalse(body.len().try_into()?),
                cur_span,
            );
            into_block.append_block(body)
        }
        StmtP::IfElse(if_expr, body_pair) => {
            let mut true_body = Block::default();
            let mut false_body = Block::default();
            compile_stmt(&body_pair.0, &mut true_body, file)?;
            compile_stmt(&body_pair.1, &mut false_body, file)?;
            true_body.add_instruction(
                Instruction::RelJump(false_body.len().try_into()?),
                cur_span.clone(),
            );
            compile_expr(if_expr, into_block, file)?;
            into_block.add_instruction(
                Instruction::RelJumpIfFalse(true_body.len().try_into()?),
                cur_span,
            );
            into_block.append_block(true_body);
            into_block.append_block(false_body);
        }
        StmtP::For(_) => todo!(),
        StmtP::Def(_) => todo!(),
        StmtP::Load(_) => todo!("load() statement unimplemented (for now)"),
    };
    Ok(())
}

pub fn compile_expr<P: AstPayload>(
    expr: &AstExprP<P>,
    into_block: &mut Block,
    file: &mut vm::BytecodeFile,
) -> Result<()> {
    let cur_span: vm::Span = (&expr.span).into();
    match &expr.node {
        ExprP::Tuple(_) => todo!(),
        ExprP::Dot(_, _) => todo!(),
        ExprP::Call(_, _) => todo!(),
        ExprP::Index(idx) => {
            compile_expr(&idx.0, into_block, file)?;
            compile_expr(&idx.1, into_block, file)?;
            into_block.add_instruction(Instruction::LoadSubscr, cur_span);
        }
        ExprP::Index2(_) => todo!(),
        ExprP::Slice(_, _, _, _) => todo!(),
        ExprP::Identifier(id) => {
            into_block.add_instruction(Instruction::LoadVar(id.ident.clone()), cur_span)
        }
        ExprP::Lambda(_) => todo!(),
        ExprP::Literal(lit) => {
            into_block.add_instruction(Instruction::PushLiteral(lit.into()), cur_span)
        }
        ExprP::Not(_) => todo!(),
        ExprP::Minus(_) => todo!(),
        ExprP::Plus(_) => todo!(),
        ExprP::BitNot(_) => todo!(),
        ExprP::Op(ex1, op, ex2) => {
            compile_expr(ex1, into_block, file)?;
            compile_expr(ex2, into_block, file)?;
            into_block.add_instruction(Instruction::BinOp(op.into()), cur_span);
        }
        ExprP::If(_) => todo!(),
        ExprP::List(_) => todo!(),
        ExprP::Dict(d) => {
            into_block.add_instruction(
                Instruction::PushLiteral(vm::value::Val::Dict(HashMap::default())),
                cur_span.clone(),
            );
            for v in d {
                let key = &v.0;
                let val = &v.1;
                compile_expr(val, into_block, file)?;
                into_block.add_instruction(Instruction::RotTwo, cur_span.clone());
                compile_expr(key, into_block, file)?;
                into_block.add_instruction(Instruction::StoreSubscr, cur_span.clone())
            }
        }
        ExprP::ListComprehension(_, _, _) => todo!(),
        ExprP::DictComprehension(_, _, _) => todo!(),
        ExprP::FString(_) => todo!(),
    };
    Ok(())
}

pub fn compile_assign<P: AstPayload>(
    expr: &AssignP<P>,
    into_block: &mut Block,
    file: &mut vm::BytecodeFile,
) -> Result<()> {
    // First, put the RHS on the stack
    compile_expr(&expr.rhs, into_block, file)?;
    // Ignore expr.ty
    // Then, assign the value
    let cur_span: vm::Span = (&expr.lhs.span).into();
    match &expr.lhs.node {
        AssignTargetP::Tuple(_) => todo!("Destructing assign not yet implemented"),
        AssignTargetP::Index(idx) => {
            compile_expr(&idx.0, into_block, file)?;
            compile_expr(&idx.1, into_block, file)?;
            into_block.add_instruction(Instruction::StoreSubscr, cur_span.clone());
            into_block.add_instruction(Instruction::Pop, cur_span);
        }
        AssignTargetP::Dot(var, prop) => {
            compile_expr(var, into_block, file)?;
            into_block.add_instruction(
                Instruction::PushLiteral(vm::value::Val::Str(prop.to_string())),
                cur_span.clone(),
            );
            into_block.add_instruction(Instruction::StoreSubscr, cur_span.clone());
            into_block.add_instruction(Instruction::Pop, cur_span);
        }
        AssignTargetP::Identifier(id) => {
            into_block.add_instruction(Instruction::StoreVar(id.node.ident.clone()), cur_span);
        }
    };
    Ok(())
}
