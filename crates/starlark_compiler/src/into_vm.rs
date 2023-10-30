use starlark_syntax::syntax::ast::{AstParameterP, AstPayload, BinOp, ParameterP};
use vm::bytecode::BinOpKind;

pub(crate) trait IntoVM<VT> {
    fn into_vm(self) -> VT;
}

impl<P: AstPayload> IntoVM<vm::BlockParameter> for &Vec<AstParameterP<P>> {
    fn into_vm(self) -> vm::BlockParameter {
        let mut bp = vm::BlockParameter::default();
        for p in self {
            match &p.node {
                ParameterP::Normal(v, _) => bp.arg_list.push(v.ident.clone()),
                ParameterP::WithDefaultValue(_, _, _) => panic!("Default values not supported"),
                ParameterP::NoArgs => continue,
                ParameterP::Args(v, _) => bp.args_name = Some(v.ident.clone()),
                ParameterP::KwArgs(v, _) => bp.kwargs_name = Some(v.ident.clone()),
            }
        }
        bp
    }
}

impl IntoVM<BinOpKind> for &BinOp {
    fn into_vm(self) -> BinOpKind {
        match self {
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
