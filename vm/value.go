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

type FloatValue float64

func (FloatValue) isValue() {}
func (f FloatValue) AsBool() bool {
	return f != 0.0
}

type StructValue map[string]Value

func (StructValue) isValue() {}
func (s StructValue) AsBool() bool {
	return len(s) > 0
}

type ArrayValue []Value

func (ArrayValue) isValue() {}
func (a ArrayValue) AsBool() bool {
	return len(a) > 0
}

type NoneValue bool

var None NoneValue = false

func (NoneValue) isValue() {}
func (NoneValue) AsBool() bool {
	return false
}
