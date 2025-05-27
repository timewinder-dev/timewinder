package vm

import "strings"

type Value interface {
	isValue()
	AsBool() bool
	Clone() Value
	Cmp(other Value) (int, bool)
}

type BoolValue bool

var (
	BoolTrue  = BoolValue(true)
	BoolFalse = BoolValue(false)
)

func (BoolValue) isValue() {}
func (b BoolValue) AsBool() bool {
	return bool(b)
}
func (b BoolValue) Clone() Value {
	return BoolValue(b)
}

func (b BoolValue) Cmp(other Value) (int, bool) {
	v, ok := other.(BoolValue)
	if !ok {
		return 0, false
	}
	if b {
		if v {
			return 0, true
		}
		return 1, true
	} else {
		if v {
			return -1, true
		}
		return 0, true
	}
}

type StrValue string

func (StrValue) isValue() {}
func (s StrValue) AsBool() bool {
	return s != ""
}
func (s StrValue) Clone() Value {
	return StrValue(s)
}
func (s StrValue) Cmp(other Value) (int, bool) {
	v, ok := other.(StrValue)
	if !ok {
		return 0, false
	}
	return strings.Compare(string(s), string(v)), true
}

type IntValue int

func (IntValue) isValue() {}
func (i IntValue) AsBool() bool {
	return i != 0
}

func (i IntValue) Clone() Value {
	return IntValue(i)
}

func (i IntValue) Cmp(other Value) (int, bool) {
	v, ok := other.(IntValue)
	if !ok {
		return 0, false
	}
	if i == v {
		return 0, true
	}
	if i < v {
		return -1, true
	}
	return 1, true
}

type FloatValue float64

func (FloatValue) isValue() {}
func (f FloatValue) AsBool() bool {
	return f != 0.0
}

func (f FloatValue) Clone() Value {
	return FloatValue(f)
}
func (f FloatValue) Cmp(other Value) (int, bool) {
	v, ok := other.(FloatValue)
	if !ok {
		return 0, false
	}
	if f == v {
		return 0, true
	}
	if f < v {
		return -1, true
	}
	return 1, true
}

type StructValue map[string]Value

func (StructValue) isValue() {}
func (s StructValue) AsBool() bool {
	return len(s) > 0
}

func (s StructValue) Clone() Value {
	out := make(map[string]Value)
	for k, v := range s {
		out[k] = v.Clone()
	}
	return StructValue(out)
}

func (s StructValue) Cmp(other Value) (int, bool) {
	return 0, false
}

type ArrayValue []Value

func (ArrayValue) isValue() {}
func (a ArrayValue) AsBool() bool {
	return len(a) > 0
}

func (a ArrayValue) Clone() Value {
	var out []Value
	for _, v := range a {
		out = append(out, v.Clone())
	}
	return ArrayValue(out)
}
func (a ArrayValue) Cmp(other Value) (int, bool) {
	return 0, false
}

type NoneValue bool

var None NoneValue = false

func (NoneValue) isValue() {}
func (NoneValue) AsBool() bool {
	return false
}

func (NoneValue) Clone() Value {
	return None
}

func (NoneValue) Cmp(other Value) (int, bool) {
	if _, ok := other.(NoneValue); ok {
		return 0, true
	}
	return 0, false
}

type ArgValue struct {
	Key   string
	Value Value
}

func (ArgValue) isValue() {}
func (a ArgValue) AsBool() bool {
	return a.Value.AsBool()
}
func (a ArgValue) Clone() Value {
	return ArgValue{
		Key:   a.Key,
		Value: a.Value.Clone(),
	}
}
func (a ArgValue) Cmp(other Value) (int, bool) {
	return 0, false
}

type FnPtrValue ExecPtr

func (FnPtrValue) isValue()       {}
func (f FnPtrValue) AsBool() bool { return true }
func (f FnPtrValue) Clone() Value {
	return FnPtrValue(f)
}
func (f FnPtrValue) Cmp(other Value) (int, bool) {
	return 0, false
}
