pub mod state;
use anyhow::Result;

use self::state::ExecutionState;

pub struct Executor {
    pub program: crate::BytecodeFile,
}

impl Executor {
    pub fn new(bc: crate::BytecodeFile) -> Self {
        Self { program: bc }
    }

    pub fn next_instruction(&self, state: &mut ExecutionState) -> Result<()> {
        state.exec_instruction(&self.program)
    }
}

#[derive(thiserror::Error, Debug)]
pub enum ExecutionError {
    #[error("Environment not ready")]
    EnvNotReady,
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn exec_state() {}
}
