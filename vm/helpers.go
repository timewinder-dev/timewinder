package vm

import (
	"io"

	"go.starlark.net/syntax"
)

func LoadFile(name string, r io.Reader) (*Program, error) {
	opts := syntax.FileOptions{}
	f, err := opts.Parse(name, r, 0)
	if err != nil {
		return nil, err
	}
	return Compile(f)
}
