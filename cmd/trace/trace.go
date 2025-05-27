package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

var (
	file = flag.String("file", "", "Source file")
	call = flag.String("call", "", "Call to make")
)

func main() {
	flag.Parse()
	if *file == "" {
		log.Fatal("--file is required")
	}
	f, err := vm.CompilePath(*file)
	if err != nil {
		log.Fatalf("couldn't compile: %s", err)
	}
	if *call != "" {
		log.Fatal("Call is unimplemented at the moment")
	}
	trace(f)
}

func trace(prog *vm.Program) {
	env := interp.NewState(1)
	g := env.Globals
	//prog.DebugPrint()
	for {
		fmt.Println("*******")
		prettyPrint(prog, g)
		v, err := interp.Step(prog, nil, []*interp.StackFrame{g})
		if err != nil {
			log.Fatalln("Got err:", err)
		}
		if v == interp.End || v == interp.Return {
			fmt.Println("Finished")
			break
		} else {
			fmt.Println("Continuing")
		}
	}
}

func prettyPrint(prog *vm.Program, f *interp.StackFrame) {
	fmt.Printf("Stack: %v\n", f.Stack)
	fmt.Printf("Variables: %v\n", f.Variables)
	inst, err := prog.GetInstruction(f.PC)
	if err != nil {
		fmt.Println("End of instructions")
	} else {
		fmt.Printf("NextOp: %s\n", inst)
	}
}
