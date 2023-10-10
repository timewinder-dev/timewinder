use anyhow::Result;

#[cfg(test)]
mod tests {

    use starlark_syntax::syntax::AstModuleFields;
    use starlark_syntax::syntax::{AstModule, Dialect};

    use super::*;

    #[test]
    fn parse_simple_string() -> Result<()> {
        let module = AstModule::parse(
            "foo.starlark",
            "x = 3\ny = 4".to_string(),
            &Dialect::Standard,
        )?;

        let stmt = module.statement();
        dbg!(stmt);
        Ok(())
    }
    #[test]
    fn parse_def() -> Result<()> {
        let module = AstModule::parse(
            "foo.starlark",
            "def foo(y):\n  x = 3 + y\n  return x\n\ng = foo(6)".to_string(),
            &Dialect::Standard,
        )?;

        let stmt = module.statement();
        dbg!(stmt);
        Ok(())
    }
}
