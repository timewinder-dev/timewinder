use anyhow::Result;

#[cfg(test)]
mod tests {

    use super::*;

    #[test]
    fn compile_add() -> Result<()> {
        let bc = crate::parse_string_to_bytecode("foo.starlark", "2 + 3".to_string())?;
        dbg!(bc);
        Ok(())
    }
}
