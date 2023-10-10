use starlark_syntax::syntax::ast::BinOp;

use crate::value::Val;

#[derive(Clone, Debug)]
pub enum Instruction {
    NoOp,
    Pop,
    PushLiteral(Val),
    TempInst(TempInstruction),
    RelJump(usize),  // delta jump
    PreCall(String), // apparent name of function
    Call,
    Return,
    BinOp(BinOpKind),
}

#[derive(Clone, Debug)]
pub enum TempInstruction {
    Continue,
    Break,
}

#[derive(Clone, Debug)]
pub enum BinOpKind {
    Or,
    And,
    Equal,
    NotEqual,
    Less,
    Greater,
    LessOrEqual,
    GreaterOrEqual,
    In,
    NotIn,
    Subtract,
    Add,
    Multiply,
    Percent,
    Divide,
    FloorDivide,
    BitAnd,
    BitOr,
    BitXor,
    LeftShift,
    RightShift,
}

impl From<&BinOp> for BinOpKind {
    fn from(value: &BinOp) -> Self {
        match value {
            BinOp::Or => BinOpKind::Or,
            BinOp::And => BinOpKind::And,
            BinOp::Equal => BinOpKind::Equal,
            BinOp::NotEqual => BinOpKind::NotEqual,
            BinOp::Less => BinOpKind::Less,
            BinOp::Greater => BinOpKind::Greater,
            BinOp::LessOrEqual => BinOpKind::LessOrEqual,
            BinOp::GreaterOrEqual => BinOpKind::GreaterOrEqual,
            BinOp::In => BinOpKind::In,
            BinOp::NotIn => BinOpKind::NotIn,
            BinOp::Subtract => BinOpKind::Subtract,
            BinOp::Add => BinOpKind::Add,
            BinOp::Multiply => BinOpKind::Multiply,
            BinOp::Percent => BinOpKind::Percent,
            BinOp::Divide => BinOpKind::Divide,
            BinOp::FloorDivide => BinOpKind::FloorDivide,
            BinOp::BitAnd => BinOpKind::BitAnd,
            BinOp::BitOr => BinOpKind::BitOr,
            BinOp::BitXor => BinOpKind::BitXor,
            BinOp::LeftShift => BinOpKind::LeftShift,
            BinOp::RightShift => BinOpKind::RightShift,
        }
    }
}
