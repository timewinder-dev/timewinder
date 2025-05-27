package cas

import (
	"errors"
	"io"
)

type CAS interface {
	Put(item Hashable) (Hash, error)
	Has(hash Hash) bool
	getReader(hash Hash) (bool, io.Reader, error)
}

type Serde interface {
	Serialize(w io.Writer) error
	Deserialize(r io.Reader) error
}

type Hashable interface {
	Serde
}

type directStore interface {
	getValue(h Hash) (bool, any, error)
}

type Hash uint64

func Retrieve[T Hashable](c CAS, hash Hash) (T, error) {
	var t T
	if v, ok := c.(directStore); ok {
		has, val, err := v.getValue(hash)
		if err != nil {
			return t, err
		}
		if has {
			return val.(T), nil
		}
	}
	return t, errors.New("Couldn't find hash in CAS")
}
