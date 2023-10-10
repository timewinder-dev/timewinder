use std::collections::HashMap;

#[derive(Clone)]
pub enum Val {
    Integer(i64),
    Float(f64),
    Str(String),
    Bool(bool),
    Dict(HashMap<Val, Val>),
    List(Vec<Val>),
}
