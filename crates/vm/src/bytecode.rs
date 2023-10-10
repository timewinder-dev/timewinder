use crate::value::Val;

#[derive(Clone)]
pub enum Instruction {
    NoOp,
    Pop,
    PushLiteral(Val),
    TempInst(TempInstruction),
    RelJump(usize),  // delta jump
    PreCall(String), // apparent name of function
    Call,
    Return,
}

#[derive(Clone)]
pub enum TempInstruction {
    Continue,
    Break,
}
