use std::collections::HashMap;

use starlark_syntax::syntax::ast::AstLiteral;

#[derive(Clone)]
pub enum Val {
    Integer(i64),
    Float(f64),
    Str(String),
    Bool(bool),
    Dict(HashMap<Val, Val>),
    List(Vec<Val>),
    Null,
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
