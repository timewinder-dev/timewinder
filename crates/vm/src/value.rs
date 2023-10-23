use anyhow::anyhow;
use anyhow::Result;
use std::collections::HashMap;

use starlark_syntax::syntax::ast::AstLiteral;

use crate::bytecode::BinOpKind;

#[derive(Clone, Debug)]
pub enum Val {
    Integer(i64),
    Float(f64),
    Str(String),
    Bool(bool),
    Dict(HashMap<String, Val>),
    List(Vec<Val>),
    Null,
}

impl Val {
    pub fn is_truthy(&self) -> bool {
        match self {
            Val::Integer(v) => v.eq(&(0)),
            Val::Float(f) => f.eq(&(0.0)),
            Val::Str(s) => s.is_empty(),
            Val::Bool(b) => *b,
            Val::Dict(d) => d.is_empty(),
            Val::List(l) => l.is_empty(),
            Val::Null => false,
        }
    }

    pub fn bin_op(&self, other: &Val, op: &BinOpKind) -> Result<Val> {
        let res = match op {
            BinOpKind::Subtract => match (self, other) {
                (Val::Integer(x), Val::Integer(y)) => Val::Integer(x - y),
                (Val::Float(x), Val::Float(y)) => Val::Float(x - y),
                _ => {
                    return Err(anyhow!(
                        "subtracting two differing types {:?} - {:?}",
                        self,
                        other
                    ))
                }
            },
            BinOpKind::Add => match (self, other) {
                (Val::Integer(x), Val::Integer(y)) => Val::Integer(x + y),
                (Val::Float(x), Val::Float(y)) => Val::Float(x + y),
                _ => {
                    return Err(anyhow!(
                        "adding two differing types {:?} + {:?}",
                        self,
                        other
                    ))
                }
            },
            BinOpKind::Multiply => todo!(),
            BinOpKind::Divide => todo!(),
            BinOpKind::Percent => todo!(),
            BinOpKind::FloorDivide => todo!(),
            BinOpKind::BitAnd => todo!(),
            BinOpKind::BitOr => todo!(),
            BinOpKind::BitXor => todo!(),
            BinOpKind::LeftShift => todo!(),
            BinOpKind::RightShift => todo!(),
            _ => return self.bin_op_comparison(other, op),
        };
        Ok(res)
    }

    pub fn bin_op_comparison(&self, other: &Val, op: &BinOpKind) -> Result<Val> {
        let res = match op {
            BinOpKind::Or => {
                if self.is_truthy() {
                    true
                } else {
                    other.is_truthy()
                }
            }
            BinOpKind::And => {
                if self.is_truthy() {
                    other.is_truthy()
                } else {
                    false
                }
            }
            BinOpKind::Equal => self.eq(other),
            BinOpKind::NotEqual => !self.eq(other),
            BinOpKind::Less => self.lt(other)?,
            BinOpKind::Greater => !self.lt(other)? && !self.eq(other),
            BinOpKind::LessOrEqual => self.lt(other)? || self.eq(other),
            BinOpKind::GreaterOrEqual => !self.lt(other)?,
            BinOpKind::In => other.contains(self),
            BinOpKind::NotIn => !other.contains(self),
            _ => panic!("Not a comparison binop kind mentioned"),
        };
        Ok(Val::Bool(res))
    }

    pub fn eq(&self, other: &Val) -> bool {
        match self {
            Val::Integer(i) => match other {
                Val::Integer(x) => i == x,
                Val::Float(f) => *f == (*i as f64),
                _ => false,
            },
            Val::Float(f) => match other {
                Val::Integer(i) => *f == (*i as f64),
                Val::Float(x) => x == f,
                _ => false,
            },
            Val::Str(s) => match other {
                Val::Str(x) => x == s,
                _ => false,
            },
            Val::Bool(b) => match other {
                Val::Bool(x) => b == x,
                _ => false,
            },
            Val::Dict(_) => todo!(),
            Val::List(l) => match other {
                Val::List(o) => {
                    if l.len() != o.len() {
                        false
                    } else {
                        l.iter()
                            .zip(o.iter())
                            .fold(true, |b, (l, r)| if !b { false } else { l.eq(r) })
                    }
                }
                _ => false,
            },
            Val::Null => match other {
                Val::Null => true,
                _ => false,
            },
        }
    }

    pub fn lt(&self, other: &Val) -> Result<bool> {
        match (self, other) {
            (Val::Integer(_), Val::Integer(_)) => todo!(),
            (Val::Integer(_), Val::Float(_)) => todo!(),
            (Val::Float(_), Val::Integer(_)) => todo!(),
            (Val::Float(_), Val::Float(_)) => todo!(),
            (Val::Str(_), Val::Str(_)) => todo!(),
            (Val::Bool(_), Val::Bool(_)) => todo!(),
            (Val::Null, Val::Null) => todo!(),
            _ => Err(anyhow!("Uncomparable types {:?} and {:?}", self, other)),
        }
    }

    pub fn contains(&self, other: &Val) -> bool {
        match (self, other) {
            (Val::Dict(_), Val::Integer(_)) => todo!(),
            (Val::Dict(_), Val::Float(_)) => todo!(),
            (Val::Dict(_), Val::Str(_)) => todo!(),
            (Val::Dict(_), Val::Bool(_)) => todo!(),
            (Val::Dict(_), Val::Null) => todo!(),
            (Val::List(_), Val::Integer(_)) => todo!(),
            (Val::List(_), Val::Float(_)) => todo!(),
            (Val::List(_), Val::Str(_)) => todo!(),
            (Val::List(_), Val::Bool(_)) => todo!(),
            (Val::List(_), Val::Dict(_)) => todo!(),
            (Val::List(_), Val::List(_)) => todo!(),
            (Val::List(_), Val::Null) => todo!(),
            _ => false,
        }
    }
}

impl From<&AstLiteral> for Val {
    fn from(value: &AstLiteral) -> Self {
        match value {
            AstLiteral::Int(ast_int) => match ast_int.node {
                starlark_syntax::lexer::TokenInt::I32(v) => Val::Integer(v.into()),
                starlark_syntax::lexer::TokenInt::BigInt(_) => {
                    panic!("BigInt literals are unsupported for now")
                }
            },
            AstLiteral::Float(ast_float) => Val::Float(ast_float.node),
            AstLiteral::String(ast_string) => Val::Str(ast_string.node.clone()),
            AstLiteral::Ellipsis => panic!("AstLiteral::Ellipsis not supported"),
        }
    }
}
