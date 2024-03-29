use crate::value::Val;

#[derive(Clone)]
pub enum Instruction {
    NoOp,
    Pop,
    PushLiteral(Val),
    AllocDict,
    AllocVec,
    TempInst(TempInstruction),
    RelJump(isize),  // delta jump
    PreCall(String), // apparent name of function
    Call,
    Return,
    BinOp(BinOpKind),  // TOS = TOS1 (op) TOS
    LoadAttr(String),  // TOS = getattr(TOS, name)
    LoadVar(String),   // TOS = env(name)
    StoreAttr(String), // TOS1.name = TOS
    StoreVar(String),  // name = TOS
    StoreSubscr,       // TOS1[TOS] = TOS2
    LoadSubscr,        // TOS = TOS1[TOS]
    RotTwo,
    RelJumpIfFalse(isize),
}

impl std::fmt::Debug for Instruction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Instruction::NoOp => f.write_str("NOP"),
            Instruction::Pop => f.write_str("POP"),
            Instruction::AllocDict => f.write_str("ALLOC_DICT"),
            Instruction::AllocVec => f.write_str("ALLOC_VEC"),
            Instruction::PushLiteral(lit) => f.write_fmt(format_args!("PUSH_LITERAL {:?}", lit)),
            Instruction::TempInst(t) => f.write_fmt(format_args!("{:?}", t)),
            Instruction::RelJump(diff) => f.write_fmt(format_args!("REL_JUMP {:?}", diff)),
            Instruction::PreCall(name) => f.write_fmt(format_args!("PRECALL {:?}", name)),
            Instruction::Call => f.write_str("CALL"),
            Instruction::Return => f.write_str("RET"),
            Instruction::BinOp(op) => f.write_fmt(format_args!("BIN_OP {:?}", op)),
            Instruction::LoadAttr(attr) => f.write_fmt(format_args!("LOAD_ATTR {:?}", attr)),
            Instruction::LoadVar(v) => f.write_fmt(format_args!("LOAD_VAR {:?}", v)),
            Instruction::StoreAttr(attr) => f.write_fmt(format_args!("STORE_ATTR {:?}", attr)),
            Instruction::StoreVar(v) => f.write_fmt(format_args!("STORE_VAR {:?}", v)),
            Instruction::StoreSubscr => f.write_str("STORE_SUBSCR"),
            Instruction::LoadSubscr => f.write_str("LOAD_SUBSCR"),
            Instruction::RotTwo => f.write_str("ROT_TWO"),
            Instruction::RelJumpIfFalse(delta) => {
                f.write_fmt(format_args!("JMP_FALSE {:?}", delta))
            }
        }
    }
}

#[derive(Clone)]
pub enum TempInstruction {
    Continue,
    Break,
}

impl std::fmt::Debug for TempInstruction {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            TempInstruction::Continue => f.write_str("CONTINUE"),
            TempInstruction::Break => f.write_str("BREAK"),
        }
    }
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
