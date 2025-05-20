package vm

type Opcode uint32

const (
	NOP Opcode = iota
	// PRE-STACK  TOS TOS+1 ... | OP |  POST-STACK |
	POP    // A | | NIL
	PUSH   // NIL | x | A
	SETVAL // A B | A = B | NIL
	GETVAL // A | retrieve B given A | B

	ADD      // A B | C = A + B | C
	SUBTRACT // A B | C = A - B | C
	MULTIPLY // A B | C = A * B | C
	DIVIDE   // A B | C = A / B | C

	EQ  // A B | C = A == B | C
	LT  // A B | C = A < B | C
	LTE // A B | C = A <= B | C
	NOT // A | B = not A | B

	JMP    // | Jumps Unconditionally to Arg |
	JFALSE // A | Jumps to Arg if A is false |

	RETURN // A | Returns A up a stack frame |

	LABEL
	OpcodeMax
)

func (o Opcode) String() string {
	switch o {
	case NOP:
		return "NOP"
	case POP:
		return "POP"
	case PUSH:
		return "PUSH"
	case SETVAL:
		return "SETVAL"
	case GETVAL:
		return "GETVAL"
	case ADD:
		return "ADD"
	case SUBTRACT:
		return "SUBTRACT"
	case MULTIPLY:
		return "MULTIPLY"
	case DIVIDE:
		return "DIVIDE"
	case LT:
		return "LT"
	case LTE:
		return "LTE"
	case EQ:
		return "EQ"
	case NOT:
		return "NOT"
	case JMP:
		return "JMP"
	case JFALSE:
		return "JFALSE"
	case RETURN:
		return "RETURN"
		// Complete all uncovered opcodes
	}
	panic("Unnamed opcode")
}
