package pswap_test

import (
	"testing"

	"github.com/mutility/parameter-swap/pswap"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, pswap.Analyzer().Analyzer, "a")

	exact := pswap.Analyzer()
	exact.ExactTypeOnly = true
	analysistest.Run(t, testdata, exact.Analyzer, "exact")
}
