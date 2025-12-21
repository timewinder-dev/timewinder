package main

import (
	"fmt"
	"go.starlark.net/syntax"
)

func main() {
	code := `
x = 1

def test():
  global x
  x = 2
`
	opts := syntax.FileOptions{}
	_, err := opts.Parse("test.star", code, 0)
	if err != nil {
		fmt.Println("Parse error:", err)
	} else {
		fmt.Println("Parsed successfully!")
	}
}
