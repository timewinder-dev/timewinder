use anyhow::Result;

#[cfg(test)]
mod tests {

    use super::*;
    use indoc::indoc;

    #[test]
    fn add_values() -> Result<()> {
        let source = indoc! {"
            f = 2 + 3
            g = 6 + f
        "};
        let bc = crate::parse_string_to_bytecode("foo.starlark", source.to_string())?;
        let exec = vm::executor::Executor::new(bc);
        let mut state = exec.make_state();
        let v = exec.run_forever(&mut state);
        v.unwrap_or_else(|e| {
            dbg!(e);
        });
        dbg!(state);
        Ok(())
    }
}
