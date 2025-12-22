# Unit Tests

This directory contains unit tests for Timewinder features that are best tested by compiling and executing small code blocks.

## Test Files

### binary_operators_test.go
Comprehensive tests for binary operators implemented for Timewinder.

**Tests included:**
- **TestModuloOperator** (5 cases)
  - Positive modulo
  - Zero remainder
  - Small mod large
  - Modulo in expressions
  - Negative modulo (Go semantics: -7 % 3 = -1, not 2)

- **TestFloorDivisionOperator** (5 cases)
  - Basic floor division
  - Exact division
  - Division with remainder
  - Zero dividend
  - Floor division in expressions

- **TestInOperatorArrays** (6 cases)
  - Element found/not found
  - First/last position
  - Empty array
  - String arrays

- **TestInOperatorStrings** (5 cases)
  - Substring found/not found
  - Single character
  - Empty substring
  - Substring at start

- **TestInOperatorDicts** (3 cases)
  - Key found/not found
  - Empty dict

- **TestNotInOperator** (6 cases)
  - Arrays, strings, dicts
  - Both true and false cases

- **TestModuloCircularIndex** (1 case)
  - Real-world use case: circular buffer indexing

- **TestInOperatorInControlFlow** (1 case)
  - Using `in` in if statements

## Running Tests

```bash
# Run all tests in this directory
go test ./test

# Run with verbose output
go test -v ./test

# Run specific test
go test -v ./test -run TestModuloOperator

# Run specific subtest
go test -v ./test -run TestModuloOperator/positive_modulo
```

## Test Results

```
PASS
ok  	github.com/timewinder-dev/timewinder/test	0.153s
```

All 32 test cases pass successfully.

## Coverage

These tests verify:
- ✅ Modulo operator (%) with positive, negative, and zero cases
- ✅ Floor division operator (//) with various inputs
- ✅ In operator (in) for arrays, strings, and dicts
- ✅ Not in operator (not in) for all collection types
- ✅ Operators in expressions and control flow
- ✅ Real-world use cases (circular indexing)

## Notes

### Go Modulo Semantics
Go's modulo operator follows truncated division, where the result takes the sign of the dividend:
- `-7 % 3 = -1` (Go behavior)
- Python would return `2`

This is important for algorithms that assume Python-like modulo behavior (e.g., negative index wrapping).

### Starlark Limitations
The `**` (power) operator is not supported by Starlark's parser, even though we implemented compiler and interpreter support. This is a language limitation, not a Timewinder limitation.
