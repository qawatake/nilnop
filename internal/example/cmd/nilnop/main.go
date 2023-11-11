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
