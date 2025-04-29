package cas

import "io"

type CAS interface {
	Put(item Hashable) (Hash, error)
	Get()
}

type Serde interface {
	Serialize(w io.Writer)
	Deserialize(r io.Reader)
}

type Hashable interface {
	Serde
	HashTo(w io.Writer)
}

type Hash uint64
