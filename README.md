# nilnop

<!-- [![Go Reference](https://pkg.go.dev/badge/github.com/qawatake/nilnop.svg)](https://pkg.go.dev/github.com/qawatake/nilnop)
[![test](https://github.com/qawatake/nilnop/actions/workflows/test.yaml/badge.svg)](https://github.com/qawatake/nilnop/actions/workflows/test.yaml)
[![codecov](https://codecov.io/gh/qawatake/nilnop/graph/badge.svg?token=0XZh5C4Gq8)](https://codecov.io/gh/qawatake/nilnop) -->

Linter `nilnop` detects nil is passed to a function that does nothing for nil.

```go
func f() (err error) {
  reportError(err) // <- nil is passed to reportError
  err = errors.New("new error")
  reportError(err) // ok because err is not nil
  return err
}

// reportError panics if err is not nil.
func reportError(err error) {
  if err != nil {
    panic(err)
  }
}
```

You can try an example by running `make run.example`.

## How to use

Build your `nilnop` binary by writing `main.go` like below.

```go
package main

import (
  "github.com/qawatake/nilnop"
  "golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
  unitchecker.Main(
    nilnop.NewAnalyzer(
      nilnop.Target{
        PkgPath:  "github.com/qawatake/nilnop/internal/example",
        FuncName: "reportError",
        ArgPos:   0,
      },
    ),
  )
}
```

Then, run `go vet` with your `nilnop` binary.

```sh
go vet -vettool=/path/to/your/nilnop ./...
```
