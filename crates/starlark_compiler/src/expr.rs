use anyhow::Result;
use starlark_syntax::syntax::ast::{AstExprP, AstPayload};
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
                None => into_block
                    .add_instruction(Instruction::PushLiteral(vm::value::Val::Null), cur_span),
            };
            into_block.add_instruction(Instruction::Return, cur_span);
        }
        StmtP::Expression(_) => todo!(),
        StmtP::Assign(_) => todo!(),
        StmtP::AssignModify(_, _, _) => todo!(),
        StmtP::Statements(_) => todo!(),
        StmtP::If(_, _) => todo!(),
        StmtP::IfElse(_, _) => todo!(),
        StmtP::For(_) => todo!(),
        StmtP::Def(_) => todo!(),
        StmtP::Load(_) => todo!(),
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
        ExprP::Index(_) => todo!(),
        ExprP::Index2(_) => todo!(),
        ExprP::Slice(_, _, _, _) => todo!(),
        ExprP::Identifier(_) => todo!(),
        ExprP::Lambda(_) => todo!(),
        ExprP::Literal(lit) => {
            into_block.add_instruction(Instruction::PushLiteral(lit.into()), cur_span)
        }
        ExprP::Not(_) => todo!(),
        ExprP::Minus(_) => todo!(),
        ExprP::Plus(_) => todo!(),
        ExprP::BitNot(_) => todo!(),
        ExprP::Op(_, _, _) => todo!(),
        ExprP::If(_) => todo!(),
        ExprP::List(_) => todo!(),
        ExprP::Dict(_) => todo!(),
        ExprP::ListComprehension(_, _, _) => todo!(),
        ExprP::DictComprehension(_, _, _) => todo!(),
        ExprP::FString(_) => todo!(),
    };
    Ok(())
}
