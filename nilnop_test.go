package nilnop_test

import (
	"testing"

	"github.com/gostaticanalysis/testutil"
	"github.com/qawatake/nilnop"
	"golang.org/x/tools/go/analysis/analysistest"
)

// TestAnalyzer is a test for Analyzer.
func TestAnalyzer(t *testing.T) {
	testdata := testutil.WithModules(t, analysistest.TestData(), nil)
	analysistest.Run(t, testdata, nilnop.NewAnalyzer(
		nilnop.Target{
			PkgPath:  "a",
			FuncName: "Wrap",
			ArgPos:   0,
		},
		nilnop.Target{
			PkgPath:  "a",
			FuncName: "S.Wrap",
			ArgPos:   0,
		},
	), "a")
}
