#[cfg(test)]
mod explore_test;
#[cfg(test)]
mod integration_test;

mod expr;

use anyhow::Result;

use starlark_syntax::syntax::AstModuleFields;
use starlark_syntax::syntax::{AstModule, Dialect};

fn string_to_astmod(filename: &str, source: String) -> Result<AstModule> {
    AstModule::parse(filename, source, &Dialect::Standard)
}

pub fn parse_string_to_bytecode(filename: &str, source: String) -> Result<vm::BytecodeFile> {
    let mut program = vm::BytecodeFile::new(filename);
    let ast = string_to_astmod(filename, source)?;
    let mut main = vm::Block::default();
    expr::compile_stmt(ast.statement(), &mut main, &mut program)?;
    let main_block_id = program.add_block(main);
    program.set_main(Some(main_block_id));
    Ok(program)
}
