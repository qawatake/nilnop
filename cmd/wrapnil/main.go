package main

import (
	"github.com/qawatake/wrapnil"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(wrapnil.Analyzer) }
