use anyhow::Result;
use starlark_syntax::syntax::ast::{AstStmt, StmtP};
use vm::bytecode::Instruction;
use vm::bytecode::TempInstruction;
use vm::Block;

pub fn compile_stmt(
    stmt: &AstStmt,
    into_block: &mut Block,
    file: &mut vm::BytecodeFile,
) -> Result<()> {
    let cur_span = &stmt.span;
    match stmt.node {
        StmtP::Break => todo!(),
        StmtP::Continue => todo!(),
        StmtP::Pass => into_block.add_instruction(Instruction::NoOp, cur_span.into()),
        StmtP::Return(_) => todo!(),
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
