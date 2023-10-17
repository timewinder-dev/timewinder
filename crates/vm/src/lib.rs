pub mod bytecode;
pub mod executor;
pub mod value;

// A direct import of the Span type from Starlark, but copied here to generalize if ever starlark
// is to be removed.
#[derive(Clone, Debug)]
pub struct Span {
    /// The position in the codemap representing the first byte of the span.
    pub begin: u32,

    /// The position after the last byte of the span.
    pub end: u32,
}

impl Span {
    pub fn new(begin: u32, end: u32) -> Self {
        Self { begin, end }
    }
}

impl From<&starlark_syntax::codemap::Span> for Span {
    fn from(value: &starlark_syntax::codemap::Span) -> Self {
        Self {
            begin: value.begin().get(),
            end: value.end().get(),
        }
    }
}

#[derive(Default, Debug)]
pub struct Block {
    instructions: Vec<bytecode::Instruction>,
    debug_locations: Vec<Option<Span>>,
}

impl Block {
    pub fn add_instruction(&mut self, inst: bytecode::Instruction, span: Span) {
        self.instructions.push(inst);
        self.debug_locations.push(Some(span));
    }

    pub fn len(&self) -> usize {
        self.instructions.len()
    }

    pub fn is_empty(&self) -> bool {
        self.instructions.is_empty()
    }

    pub fn append_block(&mut self, other: Block) {
        let total_length = other.len();
        other
            .instructions
            .into_iter()
            .enumerate()
            .map(|(i, inst)| match inst {
                bytecode::Instruction::TempInst(t) => match t {
                    bytecode::TempInstruction::Continue => todo!(),
                    bytecode::TempInstruction::Break => self.instructions.push(
                        bytecode::Instruction::RelJump((total_length - i).try_into().unwrap()),
                    ),
                },
                _ => self.instructions.push(inst),
            })
            .for_each(drop);
    }

    fn strip(&mut self) {
        self.debug_locations.clear()
    }
}

#[derive(Debug)]
pub struct BytecodeFile {
    main: Option<usize>,
    filename: String,
    pub blocks: Vec<Block>,
}

impl BytecodeFile {
    pub fn new(filename: &str) -> Self {
        Self {
            main: None,
            filename: filename.to_string(),
            blocks: Vec::new(),
        }
    }

    pub fn set_main(&mut self, main_idx: Option<usize>) {
        self.main = main_idx
    }

    pub fn filename(&self) -> &str {
        &self.filename
    }

    pub fn add_block(&mut self, block: Block) -> usize {
        let idx = self.blocks.len();
        self.blocks.push(block);
        idx
    }

    pub fn strip(&mut self) {
        self.blocks.iter_mut().for_each(|b| b.strip())
    }
}
