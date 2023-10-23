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
    pub fn lookup_var(&self, name: &String) -> Option<Val> {
        for i in (0..self.environments.len()).rev() {
            let res = self.environments[i].vals.get(name);
            if res.is_some() {
                return res.cloned();
            }
        }
        None
    }

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
            Instruction::LoadVar(varname) => {
                let val = env
                    .vals
                    .get(varname)
                    .ok_or_else(|| anyhow!("No such variable, {:?}", varname))?;
                env.stack.push(val.clone())
            }
            Instruction::StoreAttr(_) => todo!(),
            Instruction::StoreVar(varname) => {
                let val = env.stack.pop().ok_or_else(|| anyhow!("Empty stack"))?;
                env.vals.insert(varname.clone(), val);
            }
            Instruction::StoreSubscr => {
                let tos = env.stack.pop().ok_or_else(|| anyhow!("Empty stack"))?;
                let mut tos1 = env.stack.pop().ok_or_else(|| anyhow!("Empty stack"))?;
                let tos2 = env.stack.pop().ok_or_else(|| anyhow!("Empty stack"))?;
                match tos1 {
                    Val::List(ref mut l) => {
                        let off = match tos {
                            Val::Integer(i) => i,
                            _ => {
                                return Err(anyhow!(
                                    "Subscript is not an integer for this list: {:?}",
                                    tos
                                ))
                            }
                        };
                        let idx: usize = off.try_into().unwrap();
                        if l.len() <= idx {
                            return Err(anyhow!("Subscript {:?} off edge of list {:?}", off, l));
                        }
                        l[idx] = tos2;
                    }
                    Val::Dict(ref mut d) => {
                        let off = match tos {
                            Val::Str(s) => s.clone(),
                            _ => {
                                return Err(anyhow!(
                                    "Subscript is not a string for this list: {:?}",
                                    tos
                                ))
                            }
                        };
                        d.insert(off, tos2);
                    }
                    _ => {
                        return Err(anyhow!(
                            "Subscripting item is not a list or dict: {:?}",
                            tos1
                        ))
                    }
                }
                env.stack.push(tos1);
            }
            Instruction::LoadSubscr => return Err(anyhow!("loadsubscr not implemented")),
            Instruction::RotTwo => {
                let tos = env.stack.pop().ok_or_else(|| anyhow!("Empty stack"))?;
                let tos1 = env.stack.pop().ok_or_else(|| anyhow!("Empty stack"))?;
                env.stack.push(tos);
                env.stack.push(tos1);
            }
            Instruction::RelJumpIfFalse(_) => todo!(),
        };
        env.pc += 1;
        Ok(())
    }
}
