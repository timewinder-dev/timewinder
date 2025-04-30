package vm

type Opcode uint64

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

	RETURN // A | Returns A up a stack frame |

	OpcodeMax
)
