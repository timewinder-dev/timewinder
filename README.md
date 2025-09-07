# timewinder

A temporal logic model checker for Go programs.

## Overview

This project implements a model checker that can verify properties of Go programs using temporal logic specifications. It allows you to define behavioral properties and check if your program satisfies them.

## Key Components

- `cmd/timewinder/`: Command-line interface
- `model/`: Core model checking logic
- `vm/`: Virtual machine for executing Go programs
- `interp/`: Interpreter for evaluating expressions

## Features

- Load specifications from TOML files
- Run temporal logic property checks on Go programs
- Support for multi-threaded program execution
- Debug printing of program state

## How to Use

1. Create a spec file in TOML format defining your properties
2. Write a Go program that implements the behavior you want to verify
3. Run `timewinder run <specfile>` to check if the properties hold

