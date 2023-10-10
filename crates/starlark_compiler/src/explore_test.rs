use anyhow::Result;

#[cfg(test)]
mod tests {

    use super::*;
    use indoc::indoc;

    #[test]
    fn compile_add() -> Result<()> {
        let bc = crate::parse_string_to_bytecode("foo.starlark", "2 + 3".to_string())?;
        dbg!(bc);
        Ok(())
    }

    #[test]
    fn compile_assign_lines() -> Result<()> {
        let source = indoc! {"
            f = 2 + 3
            g = 6 + f
        "};
        let mut bc = crate::parse_string_to_bytecode("foo.starlark", source.to_string())?;
        bc.strip();
        dbg!(bc);
        Ok(())
    }

    #[test]
    fn compile_assign_into_dict() -> Result<()> {
        let source = indoc! {"
            dict = {\"baz\": \"woo\"}
            dict[\"foo\"] = 6
            dict[\"bar\"] = dict[\"foo\"] + 3
        "};
        let mut bc = crate::parse_string_to_bytecode("foo.starlark", source.to_string())?;
        bc.strip();
        dbg!(bc);
        Ok(())
    }

    #[test]
    fn compile_def() -> Result<()> {
        let source = indoc! {"
            def f(x):
                return x + 3
            g = f(6)
        "};
        let mut bc = crate::parse_string_to_bytecode("foo.starlark", source.to_string())?;
        bc.strip();
        dbg!(bc);
        Ok(())
    }
}
