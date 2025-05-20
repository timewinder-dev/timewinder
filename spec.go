package timewinder

import (
	"io"

	"github.com/BurntSushi/toml"
)

type Spec struct {
	Spec       SpecDetails         `toml:",omitempty"`
	Properties map[string]Property `toml:",omitempty"`
}

type SpecDetails struct {
	File       string `toml:",omitempty"`
	Entrypoint string `toml:",omitempty"`
}

type Property struct {
	Always           string `toml:",omitempty"`
	Eventually       string `toml:",omitempty"`
	EventuallyAlways string `toml:",omitempty"`
	AlwaysEventually string `toml:",omitempty"`
}

func ParseSpec(f io.Reader) (*Spec, error) {
	var out Spec
	_, err := toml.NewDecoder(f).Decode(&out)
	return &out, err
}
