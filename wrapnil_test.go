package wrapnil_test

import (
	"testing"

	"github.com/gostaticanalysis/testutil"
	"github.com/qawatake/wrapnil"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	analysistest.Run(t, testdata, wrapnil.NewAnalyzer(
		wrapnil.Target{
			PkgPath:  "a",
			FuncName: "Wrap",
			ArgPos:   0,
		},
		wrapnil.Target{
			PkgPath:  "a",
			FuncName: "S.Wrap",
			ArgPos:   0,
		},
	), "a")
}
