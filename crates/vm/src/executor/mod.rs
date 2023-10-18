pub mod state;
use anyhow::Result;

use crate::bytecode::Instruction;

use self::state::ExecutionState;

type ExecutionResult<T> = std::result::Result<T, ExecutionError>;

pub struct Executor {
    pub program: crate::BytecodeFile,
}

impl Executor {
    pub fn new(bc: crate::BytecodeFile) -> Self {
        Self { program: bc }
    }

    pub fn make_state(&self) -> crate::executor::ExecutionState {
        let mut state = crate::executor::ExecutionState::default();
        let env = crate::executor::state::Environment {
            block: self.program.main.unwrap_or(0),
            ..Default::default()
        };
        state.environments.push(env);
        state
    }

    pub fn run_forever(&self, state: &mut ExecutionState) -> ExecutionResult<()> {
        self.run_until(state, |_| false)
    }

    pub fn run_until<F>(&self, state: &mut ExecutionState, closure: F) -> ExecutionResult<()>
    where
        F: Fn(&Instruction) -> bool,
    {
        loop {
            let inst = self.run_one_instruction(state)?;
            if closure(inst) {
                break;
            }
        }
        Ok(())
    }

    pub fn next_instruction(&self, state: &mut ExecutionState) -> ExecutionResult<()> {
        let _ = self.run_one_instruction(state)?;
        Ok(())
    }

    fn run_one_instruction(&self, state: &mut ExecutionState) -> ExecutionResult<&Instruction> {
        let pc = state
            .get_pc()
            .ok_or(ExecutionError::GenericError(anyhow::anyhow!("Invalid PC")))?;
        let block = &self.program.blocks[pc.0];
        let inst = &block
            .instructions
            .get(pc.1)
            .ok_or(ExecutionError::ExecutionComplete)?;
        state
            .exec_instruction(inst)
            .map_err(ExecutionError::GenericError)?;
        Ok(inst)
    }
}

#[derive(thiserror::Error, Debug)]
pub enum ExecutionError {
    #[error("Environment not ready")]
    EnvNotReady,
    #[error("Execution complete")]
    ExecutionComplete,
    #[error("General Execution Error: {0}")]
    GenericError(#[source] anyhow::Error),
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn exec_state() {}
}
