# Parameter Swap

`parameter-swap` is a
[go/analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis)-based tool that
identifies mismatches between named arguments and named function parameters.
This may be legit, but is often a cause for concern.

[![CI](https://github.com/mutility/parameter-swap/actions/workflows/build.yaml/badge.svg)](https://github.com/mutility/parameter-swap/actions/workflows/build.yaml)

## Example messages

Given the following source code `example.go`:

```go
     1  package example
     2
     3  func foo(one, two int) int {
     4      return one + two
     5  }
     6
     7  func main() {
     8      one, two := 1, 2
     9      foo(two, one)
    10  }
```

parameter-swap will report the following:

```console
$ parameter-swap ./...
.../example.go:9:6: passes 'two' as 'one' in call to func <PATH>.foo(one int, two int) int (position 0 vs 1)
.../example.go:9:11: passes 'one' as 'two' in call to func <PATH>.foo(one int, two int) int (position 1 vs 0)
exit status 3
```

## Usage

Run from source with `go run github.com/mutility/parameter-swap@latest` or
install with `go install github.com/mutility/parameter-swap@latest` and run
parameter-swap from GOPATH/bin.

You can configure behvior at the command line by passing the flags below, or in
library use by setting fields on `pswap.Analyzer()`.

Flag | Field | Meaning
-|-|-
`-exact` | ExactTypeOnly | Suppress reports of mismatched parameters of mismatching types
`-gen` | IncludeGeneratedFiles | Include reports from generated files

## Bug reports and feature contributions

`parameter-swap` is developed in spare time, so while bug reports and feature
contributions are welcomed, it may take a while for them to be reviewed. If
possible, try to find a minimal reproduction before reporting a bug. Bugs that
are difficult or impossible to reproduce will likely be closed.

All bug fixes will include tests to help ensure no regression; correspondingly
all contributions should include such tests.

## Mutility Analyzers

`parameter-swap` is part of [mutility-analyzers](https://github.com/mutility/analyzers).
