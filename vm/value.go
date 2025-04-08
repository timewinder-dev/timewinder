package vm

type Value interface {
	isValue()
	AsBool() bool
}

type BoolValue bool

func (BoolValue) isValue() {}

var (
	BoolTrue  = BoolValue(true)
	BoolFalse = BoolValue(false)
)

func (b BoolValue) AsBool() bool {
	return bool(b)
}

type StrValue string

func (StrValue) isValue() {}
func (s StrValue) AsBool() bool {
	return s != ""
}

type IntValue int

func (IntValue) isValue() {}
func (i IntValue) AsBool() bool {
	return i != 0
}

type FloatValue int

func (FloatValue) isValue() {}

type StructValue map[string]Value

func (StructValue) isValue() {}

type ArrayValue []Value

func (ArrayValue) isValue() {}
