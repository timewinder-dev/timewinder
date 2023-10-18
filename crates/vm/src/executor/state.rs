use anyhow::{anyhow, Result};
use std::collections::HashMap;

use crate::{
    bytecode::{BinOpKind, Instruction},
    value::Val,
};

#[derive(Default, Debug)]
pub struct Environment {
    pub vals: HashMap<String, Val>,
    pub stack: Vec<Val>,
    pub precall: Vec<String>,
    pub block: usize,
    pub pc: usize,
}

impl Environment {
    fn apply_binop(&mut self, op: &BinOpKind) -> Result<()> {
        let rhs = self
            .stack
            .pop()
            .ok_or(anyhow!("Ran out of stack for lhs in binop"))?;
        let lhs = self
            .stack
            .pop()
            .ok_or(anyhow!("Ran out of stack for lhs in binop"))?;
        let val = lhs.bin_op(&rhs, op)?;
        self.stack.push(val);
        Ok(())
    }
}

#[derive(Default, Debug)]
pub struct ExecutionState {
    pub environments: Vec<Environment>,
}

impl ExecutionState {
    pub(crate) fn get_pc(&self) -> Option<(usize, usize)> {
        self.environments.last().map(|l| (l.block, l.pc))
    }

    fn last_env(&mut self) -> Result<&mut Environment> {
        match self.environments.last_mut() {
            Some(x) => Ok(x),
            None => Err(crate::executor::ExecutionError::EnvNotReady.into()),
        }
    }
    pub(crate) fn exec_instruction(&mut self, instruction: &Instruction) -> Result<()> {
        let env = self.last_env()?;
        match instruction {
            Instruction::NoOp => {}
            Instruction::Pop => {
                let v = env.stack.pop();
                if v.is_none() {
                    return Err(anyhow!("Stack is empty"));
                }
            }
            Instruction::PushLiteral(lit) => env.stack.push(lit.clone()),
            Instruction::TempInst(_) => {
                return Err(anyhow!(
                    "Temporary Instruction encountered during execution"
                ))
            }
            Instruction::RelJump(off) => {
                env.pc = env
                    .pc
                    .checked_add_signed(*off)
                    .ok_or(anyhow!("Relative jump off the edge"))?;
                return Ok(());
            }
            Instruction::PreCall(name) => env.precall.push(name.clone()),
            Instruction::Call => todo!(),
            Instruction::Return => todo!(),
            Instruction::BinOp(op) => env.apply_binop(op)?,
            Instruction::LoadAttr(_) => todo!(),
            Instruction::LoadVar(_) => todo!(),
            Instruction::StoreAttr(_) => todo!(),
            Instruction::StoreVar(_) => todo!(),
            Instruction::StoreSubscr => todo!(),
            Instruction::LoadSubscr => todo!(),
            Instruction::RotTwo => todo!(),
            Instruction::RelJumpIfFalse(_) => todo!(),
        };
        env.pc += 1;
        Ok(())
    }
}
