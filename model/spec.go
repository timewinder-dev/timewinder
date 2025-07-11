package model

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/timewinder-dev/timewinder/vm"
)

type Spec struct {
	Spec       SpecDetails             `toml:""`
	Threads    map[string]ThreadSpec   `toml:",omitempty"`
	Properties map[string]PropertySpec `toml:",omitempty"`
}

type SpecDetails struct {
	File string `toml:",omitempty"`
}

type ThreadSpec struct {
	Entrypoint string `toml:",omitempty"`
}

type PropertySpec struct {
	Always           string `toml:",omitempty"`
	Eventually       string `toml:",omitempty"`
	EventuallyAlways string `toml:",omitempty"`
	AlwaysEventually string `toml:",omitempty"`
}

func parseSpec(f io.Reader) (*Spec, error) {
	var out Spec
	_, err := toml.NewDecoder(f).Decode(&out)
	return &out, err
}

func LoadSpecFromFile(path string) (*Spec, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	s, err := parseSpec(f)
	if err != nil {
		return nil, err
	}
	if s.Spec.File == "" {
		parts := strings.Split(fi.Name(), ".")
		parts = parts[:len(parts)-1]
		parts = append(parts, "star")
		s.Spec.File = strings.Join(parts, ".")
	}
	filedir := filepath.Dir(path)
	s.Spec.File = filepath.Clean(filepath.Join(filedir, s.Spec.File))
	return s, nil
}

func (s *Spec) BuildExecutor() (*Executor, error) {
	p, err := vm.CompilePath(s.Spec.File)
	if err != nil {
		return nil, err
	}
	exec := &Executor{
		Program: p,
		Spec:    s,
	}
	return exec, nil
}
