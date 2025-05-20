package timewinder

type Spec struct {
	Spec       SpecDetails         `yaml:",omitempty"`
	Properties map[string]Property `yaml:",omitempty"`
}

type SpecDetails struct {
	File       string `yaml:",omitempty"`
	Entrypoint string `yaml:",omitempty"`
}

type Property struct {
	Always           string `yaml:",omitempty"`
	Eventually       string `yaml:",omitempty"`
	EventuallyAlways string `yaml:",omitempty"`
	AlwaysEventually string `yaml:",omitempty"`
}
