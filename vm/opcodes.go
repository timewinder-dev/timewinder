package vm

type Opcode uint32

const (
	NOP Opcode = iota
	// PRE-STACK ... TOS+1 TOS | OP |  POST-STACK |
	POP     // A | | NIL
	PUSH    // NIL | x | A
	SETVAL  // A B | B = A | NIL
	GETVAL  // A | retrieve B given A | B
	GETATTR // A B | C = A[B] | C
	SETATTR // C A B | A[B] = C |
	SWAP    // A B | | B A
	DUP     // A | | A A

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

	BUILD_LIST // A B C | 3 | [A B C]
	BUILD_DICT // [A B] [C D] | 2 | {A: B, C: D}
	BUILD_ARG  // A | name | ARG(name, A)

	ITER_START   // X IT | Pushes to iterator stack, arg is the end label |
	ITER_START_2 // X Y IT | Pushes to iterator stack, arg is the end label |
	ITER_NEXT    // Nexts the iteration
	ITER_END     // Pops the iterator stack prematurely, jumps to end label

	CALL // A B C Fn | arg: 3, calls Fn with the top three args |

	// Here begin the opcodes that are unique to a VM that is trying to run through a search. They should add a value to the stack, but are hints to the execution.
	YIELD      // Arg: step name. Pauses execution and maybe something else runs. Breaks atomicity of actions in a function.
	FAIR_YIELD // Arg: step name. Weakly fair yield (from fstep) - pauses but no stutter checking.

	LABEL
	OpcodeMax
)

func (o Opcode) String() string {
	switch o {
	// Complete the switch with all the opcodes, including the ones that are missing
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
	case BUILD_LIST:
		return "BUILD_LIST"
	case BUILD_DICT:
		return "BUILD_DICT"
	case BUILD_ARG:
		return "BUILD_ARG"
	case ITER_START:
		return "ITER_START"
	case ITER_START_2:
		return "ITER_START_2"
	case ITER_NEXT:
		return "ITER_NEXT"
	case ITER_END:
		return "ITER_END"
	case LABEL:
		return "LABEL"
	case SWAP:
		return "SWAP"
	case DUP:
		return "DUP"
	case CALL:
		return "CALL"
	case GETATTR:
		return "GETATTR"
	case SETATTR:
		return "SETATTR"
	case YIELD:
		return "YIELD"
	case FAIR_YIELD:
		return "FAIR_YIELD"
		// Complete all uncovered opcodes
	}
	panic("Unnamed opcode")
}
